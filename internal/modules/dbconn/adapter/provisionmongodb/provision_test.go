package provisionmongodb

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

var errDriverFixture = errors.New("driver failure fixture")

func TestMongoUserCommandsUseOnlyReadWriteProjectRole(t *testing.T) {
	name, password := "p_123", "generated123"
	wantRoles := bson.A{bson.D{{Key: "role", Value: "readWrite"}, {Key: "db", Value: name}}}
	for _, test := range []struct {
		verb string
		got  bson.D
	}{
		{got: createUserCommand(name, password), verb: "createUser"},
		{got: updateUserCommand(name, password), verb: "updateUser"},
	} {
		want := bson.D{{Key: test.verb, Value: name}, {Key: "pwd", Value: password}, {Key: "roles", Value: wantRoles}}
		if !reflect.DeepEqual(test.got, want) {
			t.Errorf("command = %#v, want %#v", test.got, want)
		}
	}
	if domain.ValidDatabaseIdentifier("x;drop") {
		t.Fatal("hostile project identifier was accepted")
	}
}

func TestProvisionUpdatesExistingUserAndUsesSavedTLSSettings(t *testing.T) {
	settings := domain.SharedSettings{
		Kind: "mongodb", Host: "db.example", Port: 27017,
		Database: "workspace", User: "admin", Password: "admin-secret", TLS: "ca",
	}
	fake := &fakeCommands{duplicate: true}
	adapter := New(settings)
	adapter.connect = func(opts *options.ClientOptions) (commandRunner, error) {
		if opts.Auth == nil || opts.Auth.Username != "admin" || opts.Auth.AuthSource != "workspace" ||
			opts.TLSConfig == nil || opts.TLSConfig.ServerName != "db.example" {
			t.Fatal("Mongo client did not receive admin credentials and TLS settings")
		}
		if !strings.Contains(opts.GetURI(), "authSource=workspace") {
			t.Fatal("admin URI did not authenticate against the saved database")
		}
		return fake, nil
	}
	first, second := provisionTwice(t, adapter)
	assertMongoReconciled(t, fake, first, second, settings)
}

func provisionTwice(t *testing.T, adapter *Provisioner) (application.ProvisionResult, application.ProvisionResult) {
	t.Helper()
	first, err := adapter.Provision(context.Background(), "project-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Provision(context.Background(), "project-a")
	if err != nil {
		t.Fatal(err)
	}
	return first, second
}

func assertMongoReconciled(
	t *testing.T, fake *fakeCommands, first, second application.ProvisionResult,
	settings domain.SharedSettings,
) {
	t.Helper()
	if len(fake.commands) != 4 || fake.commands[0][0].Key != "createUser" ||
		fake.commands[1][0].Key != "updateUser" || fake.commands[2][0].Key != "createUser" ||
		fake.commands[3][0].Key != "updateUser" || first.Database != second.Database ||
		first.Host != settings.Host || first.Port != settings.Port || !first.TLS {
		t.Fatalf("unexpected Mongo reconciliation: %#v %#v %#v", first, second, fake.commands)
	}
}

func TestProvisionMapsErrorsWithoutLeakingThem(t *testing.T) {
	secret := strings.Join([]string{"mongodb://admin:", "admin-secret", "@db.example/admin"}, "")
	for _, test := range []struct {
		err  error
		want application.ProvisionFailure
	}{
		{err: mongo.CommandError{Code: mongoAuthenticationFailed, Message: secret}, want: application.AdminRejected},
		{err: mongo.CommandError{Code: mongoUnauthorized, Message: secret}, want: application.InsufficientPrivileges},
		{err: errDriverFixture, want: application.ProvisionFailed},
	} {
		got := classifyError(test.err).Error()
		if got != string(test.want) || strings.Contains(got, "admin-secret") {
			t.Fatalf("unsafe error = %q, want %q", got, test.want)
		}
	}
}

type fakeCommands struct {
	commands  []bson.D
	duplicate bool
}

func (f *fakeCommands) Run(_ context.Context, database string, command bson.D) error {
	if !domain.ValidDatabaseIdentifier(database) {
		return domain.ErrInvalidDatabaseIdentifier
	}
	f.commands = append(f.commands, command)
	if f.duplicate && command[0].Key == "createUser" {
		return mongo.CommandError{Code: 51003}
	}
	return nil
}

func (*fakeCommands) Close(context.Context) error { return nil }

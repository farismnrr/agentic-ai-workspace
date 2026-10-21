package provisionmongodb

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func reconcileUser(ctx context.Context, client commandRunner, name, password string) error {
	if err := client.Run(ctx, name, createUserCommand(name, password)); err != nil {
		var command mongo.CommandError
		if !asCommandError(err, &command) || command.Code != 51003 {
			return fmt.Errorf("create MongoDB project user: %w", err)
		}
		if err := client.Run(ctx, name, updateUserCommand(name, password)); err != nil {
			return fmt.Errorf("update MongoDB project user: %w", err)
		}
		return nil
	}
	return nil
}

func createUserCommand(name, password string) bson.D {
	return userCommand("createUser", name, password)
}

func updateUserCommand(name, password string) bson.D {
	return userCommand("updateUser", name, password)
}

func userCommand(action, name, password string) bson.D {
	roles := bson.A{bson.D{{Key: "role", Value: "readWrite"}, {Key: "db", Value: name}}}
	return bson.D{{Key: action, Value: name}, {Key: "pwd", Value: password}, {Key: "roles", Value: roles}}
}

func generatedPassword() (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, generatedPasswordLength)
	for index := range password {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate MongoDB project password: %w", err)
		}
		password[index] = alphabet[value.Int64()]
	}
	return string(password), nil
}

func asCommandError(err error, target *mongo.CommandError) bool {
	return errors.As(err, target)
}

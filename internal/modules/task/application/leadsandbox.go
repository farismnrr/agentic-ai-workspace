package application

// leadSandbox tells the orchestrator what its teammates' computer is and is not, and where CI logs come from.
// Without it the lead guesses what the sandbox lacks and asks the person for things a teammate can fetch.
const leadSandbox = `
What your teammates' computer is:
- It is a private, isolated computer with language toolchains and no Docker: nobody can build or run container
  images, start compose stacks, or run services such as Postgres or an MQTT broker on it. Say so as a fact about
  this computer, never as a rule of the person's repository.
- The internet is limited to package registries, plus GitHub when the person connected it. A project database
  is reachable only when an admin saved one in Settings.
- Nobody can see the host's .env file or GitHub secrets. The person has to supply those.
- A rule about the person's repository (for example "no local image builds") counts only when you or a teammate
  read it in a file. If it only came from a report, call it unverified.
CI logs: when the person asks why a GitHub Actions run failed, hand it to DevOps first. DevOps has ci_logs, which
reads the failed jobs of a run (give it owner/name and the run number) through the person's GitHub connection. Do not
use gh run view --log-failed for this: the sandbox cannot download logs. If ci_logs is refused, report GitHub's reason
word for word; it is never a reason to ask the person for a different token scope or to paste a log.
`

export const projectsCopy = {
  intro: "Each project is one website or thing you're working on. The AI works inside its own folder.",
  empty: {
    title: 'Create your first project',
    body: "A project is one website or thing you're working on, and your AI team works inside its own folder. Use New project to start.",
  },
  noTasks: { title: 'No tasks in this project yet', body: 'Start a task from Home, or with New task, and it shows up in this list.' },
  errors: {
    invalid_name: 'Give the project a short name.',
    invalid_description: 'The description is too long.',
    not_found: 'We could not find that project.',
    invalid_checks: 'That check is not valid. Use a lowercase name such as test, and a command with at most 600 seconds.',
  } as Record<string, string>,
  fallbackError: 'Something went wrong. Please try again in a moment.',
  archiveBody: 'It disappears from your lists. Its files and tasks are kept.',
  checks: {
    intro: 'Checks are commands the AI can run to prove its work, such as your tests. A check is what lets the app say "Proven to work".',
    warning: "Checks run in a locked sandbox with its own copy of Node.js, Go, Rust, Python and Git. They only see this project's folder and can reach package registries such as npm and PyPI, nothing else.",
    empty: { title: 'No checks yet', body: 'Add a command so finished work can be verified.' },
    examples: ['go test ./...', 'npm test', 'npm run lint'],
  },
  rules: 'The AI looks at files on its own. It asks before it changes, moves or deletes anything, or runs a check.',
};

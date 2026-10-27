# Browser extensions

Ask the AI to build a browser extension, then open **Settings → Workspace → Extensions** to install it.

## How it works

- Ask for it in plain words, such as "a button that saves the page I am reading".
- The AI first asks what sites it should work on and what it may read.
- It builds the extension, checks it, and DevOps packages it.
- The thread shows "Your extension is ready to install". Press **Install**.
- You can also open Extensions with Ctrl/Cmd+K. Sites are web apps; extensions run in your browser.

## Install in Chrome or Edge

1. Download the package and unzip it. Keep the folder.
2. Open `chrome://extensions` (or `edge://extensions`). Copy the address; a link cannot open it.
3. Turn on **Developer mode**.
4. Press **Load unpacked** and choose the folder you unzipped.
5. Pin it from the puzzle piece in the toolbar.

## What you are told before you install

- Each extension lists what it can do, in plain words, such as "Read your browsing history".
- Things that need a lot of trust are marked **Needs trust**.
- The AI is told to ask for as few permissions as it can.
- The security teammate checks the extension harder than a web app.

## Updating

- When the AI deploys a new version, download it again.
- Unzip it over the same folder, then press the refresh icon on the extension in the same page.

## Limits

- Chrome and Edge only (manifest version 3). Firefox may come later.
- Nothing is sent to a store, so there is no review and no store account.
- Chrome and Edge may warn that the extension is not from their store. That is expected.
- The browser itself is not run by the server, so the AI cannot click through the extension. It checks the build and the popup page.
- A package is limited to 3000 files and 50 MB.

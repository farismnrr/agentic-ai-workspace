// Draws a preview picture of every style family of the Designer's library, by filling scripts/style-preview.html with
// the family's own tokens.css. The pictures go to web/src/features/workspace/assets/style-previews/ and are
// committed, so the app does not draw them at run time. Needs Playwright installed globally:
//   npm i -g playwright && npx playwright install chromium
// Usage: PW=$(npm root -g)/playwright node scripts/style-previews.ts [style ...]
import { mkdirSync, readFileSync, readdirSync } from 'node:fs';
import { createRequire } from 'node:module';

const { chromium } = createRequire(import.meta.url)(process.env.PW as string);
const library = 'internal/modules/agentteam/domain/skills/library/style';
const output = 'web/src/features/workspace/assets/style-previews';
const template = readFileSync('scripts/style-preview.html', 'utf8');
// The families that look clearly different from one another. The rest of the library stays for the Designer, but
// several are twins of these (clean of minimal, elegant and editorial of warm-editorial, neumorphism of
// claymorphism) and a form of lookalike pictures only confuses. `modern` is left out because its accent is the
// indigo that marks a page as machine-made.
const offered = ['warm-editorial', 'minimal', 'corporate', 'friendly', 'brutalism', 'retro', 'glassmorphism', 'claymorphism'];
const wanted = process.argv.slice(2);
const styles = readdirSync(library, { withFileTypes: true })
  .filter((entry) => entry.isDirectory() && (wanted.length === 0 ? offered : wanted).includes(entry.name))
  .map((entry) => entry.name);

mkdirSync(output, { recursive: true });
const browser = await chromium.launch();
// A scale of 0.625 turns the 1280x800 page into an 800x500 picture, enough for a card in a form.
const page = await (await browser.newContext({ viewport: { width: 1280, height: 800 }, deviceScaleFactor: 0.625 })).newPage();

for (const style of styles) {
  const tokens = readFileSync(`${library}/${style}/tokens.css`, 'utf8');
  await page.setContent(template.replace('/*TOKENS*/', tokens), { waitUntil: 'load' });
  await page.screenshot({ path: `${output}/${style}.jpg`, type: 'jpeg', quality: 82 });
  console.log(`drew ${style}`);
}

await browser.close();

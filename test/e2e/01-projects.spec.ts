// Browser black-box coverage of the current New Project wizard, not the removed legacy dialog.
import { readFileSync } from 'node:fs';
import { createRequire } from 'node:module';

const { chromium } = createRequire(import.meta.url)(process.env.PW as string) as
  typeof import('../../web/node_modules/playwright');
const info = JSON.parse(readFileSync(process.argv[2] as string, 'utf8')) as {base:string;path:string;email:string;password:string};
const shots = process.argv[3] as string;
let failures = 0;
function check(label:string, pass:boolean, detail=''):void {
  console.log((pass ? 'PASS' : 'FAIL') + ' ' + label + (detail ? ' — ' + detail : ''));
  if (!pass) failures++;
}

const browser = await chromium.launch({ executablePath:'/usr/bin/chromium-headless-shell', args:['--no-sandbox'] });
try {
  const page = await (await browser.newContext({viewport:{width:1280,height:900}})).newPage();
  const errors:string[] = [];
  page.on('pageerror', (e) => errors.push(e.message));
  let github:'off'|'connect'|'ready'|'error' = 'off';
  const repos=Array.from({length:60},(_,i)=>({name:`ann/${i%7===0?'shop':'repo'}-${String(i).padStart(2,'0')}`,description:'',private:i%5===0}));
  let imported='';
  await page.route('**/api/integrations/github', (route) => {
    if (github==='error') return route.fulfill({status:500,json:{error:'internal'}});
    return route.fulfill({json:{configured:github!=='off',connected:github==='ready',login:'ann',name:'Ann',reposUrl:''}});
  });
  await page.route('**/api/integrations/github/repos', route =>
    route.request().method()==='POST'
      ? route.fulfill({status:201,json:{name:'ann/new-shop',private:true}})
      : route.fulfill({json:{repos}}));
  await page.route('**/api/projects/*/import', route => {
    imported=JSON.stringify(route.request().postDataJSON());
    return route.fulfill({status:204,body:''});
  });
  await page.goto(info.base+'/login');
  await page.getByLabel(/email/i).fill(info.email);
  await page.locator('#password').fill(info.password);
  await page.getByRole('button',{name:/sign in/i}).click();
  await page.waitForURL(/\/app/, {timeout:15000});

  for (const state of ['off','error'] as const) {
    github=state;
    await page.goto(info.base+'/app/projects/new');
    await page.getByRole('heading',{name:'Pick a template'}).waitFor();
    check('GitHub '+state+' skips source step', await page.getByRole('button',{name:/Without a template/}).isVisible());
    await page.getByRole('button',{name:/Without a template/}).click();
    await page.getByRole('heading',{name:'Where should it live?'}).waitFor();
    await page.getByRole('button',{name:/New project/}).click();
    await page.locator('#project-name').waitFor();
    check('details are a single section card', await page.locator('form.rounded-xl').count()===1);
    check('blank details do not allow creation', await page.getByRole('button',{name:'Create project'}).isDisabled());
    await page.getByRole('button',{name:'Cancel'}).click();
    await page.waitForURL(/\/app\/projects$/);
  }

  github='connect';
  await page.goto(info.base+'/app/projects/new');
  await page.getByRole('heading',{name:'How do you want to start?'}).waitFor();
  check('disconnected GitHub is explained',await page.getByRole('link',{name:'Connect GitHub'}).getAttribute('href')==='/app/settings/integrations');
  await page.getByRole('button',{name:/From a template/}).click();
  check('template gallery is reachable',await page.getByRole('heading',{name:'Pick a template'}).isVisible());

  github='ready';
  await page.goto(info.base+'/app/projects/new');
  await page.getByRole('heading',{name:'How do you want to start?'}).waitFor();
  await page.getByRole('button',{name:/From a GitHub repository/}).click();
  await page.getByRole('heading',{name:'Project details'}).waitFor();
  const list=page.getByRole('listbox');
  await list.waitFor();
  check('repo picker scrolls without overflowing',await list.evaluate(el=>el.scrollHeight>el.clientHeight && el.clientHeight<220));
  await page.getByRole('combobox').fill('shop-0');
  const count=await list.getByRole('option').count();
  check('repo search filters',count>0 && count<10,`${count} shown`);
  await page.getByRole('combobox').fill('zz-no-result');
  check('empty repo search explains why', (await list.innerText()).includes('No repository'));
  await page.getByRole('combobox').fill('shop-14');
  await list.getByRole('button').first().click();
  check('repo selection prefills project',await page.locator('#project-name').inputValue()==='shop-14');
  await page.screenshot({path:shots+'/new-project-github-current.png'});
  await page.getByText(/Available\. Your app will be at/).waitFor({timeout:10000});
  await page.getByRole('button',{name:'Create and import'}).click();
  await page.waitForURL(url=>url.pathname.startsWith('/app/projects/') && url.pathname!=='/app/projects/new', {timeout:15000});
  check('selected GitHub repo is imported', imported.includes('ann/shop-14'), imported);

  github='ready';
  await page.goto(info.base+'/app/projects/new');
  await page.getByRole('heading',{name:'How do you want to start?'}).waitFor();
  await page.getByRole('button',{name:/From a template/}).click();
  await page.getByRole('button',{name:/Without a template/}).click();
  await page.getByRole('button',{name:/New repository/}).click();
  await page.locator('#project-name').fill('Warung Kopi');
  await page.locator('#project-repo-name').fill('kopi-site');
  await page.getByText(/Available\. Your app will be at/).waitFor({timeout:10000});
  await page.getByRole('button',{name:'Create repository and project'}).click();
  await page.waitForURL(url=>url.pathname.startsWith('/app/projects/') && url.pathname!=='/app/projects/new', {timeout:15000});
  check('new GitHub repository creates and imports project', imported.includes('ann/new-shop'), imported);
  // Checks: empty state, native dialog, validation, save/edit and destructive confirmation.
  await page.goto(info.base + info.path);
  await page.getByRole('tab', { name:'Checks' }).click();
  const checks = page.getByRole('region', { name:'Checks' });
  await checks.getByRole('button', { name:'Add a check' }).click();
  const modal = page.getByRole('dialog', { name:'Add a check' });
  await modal.waitFor();
  check('check modal uses one accessible dialog', await modal.count()===1 && await modal.locator('.modal-box').count()===1);
  await page.keyboard.press('Escape');
  await modal.waitFor({state:'hidden',timeout:1500}).catch(() => undefined);
  check('Escape dismisses check modal', !await modal.isVisible(), JSON.stringify(await page.locator('dialog').first().evaluate(el=>({open:(el as HTMLDialogElement).open,active:document.activeElement?.tagName}))));
  await checks.getByRole('button', { name:'Add a check' }).click();
  await page.locator('#check-name').fill('lint');
  await page.locator('#check-command').fill('echo verified');
  await modal.getByRole('button', { name:'Save check' }).click();
  await checks.getByRole('button', { name:'Edit' }).waitFor();
  check('creating a check replaces empty state with flat row', (await checks.getByRole('button', { name:'Edit' }).count())===1);
  await checks.getByRole('button',{name:'Edit'}).click();
  await page.getByRole('dialog',{name:'Edit check'}).waitFor();
  await page.locator('#check-command').fill('echo checked');
  await page.getByRole('button',{name:'Save check'}).click();
  await checks.getByText('echo checked').waitFor();
  check('edit check updates row command',await checks.getByText('echo checked').count()===1);
  await checks.getByRole('button',{name:'Remove lint'}).click();
  await page.getByRole('dialog').getByRole('button',{name:'Remove check'}).click();
  await checks.getByRole('button',{name:'Add a check'}).waitFor();
  check('remove returns to one empty state CTA',await checks.getByRole('button',{name:'Add a check'}).count()===1);
  for (const width of [390,820,1024,1440]) {
    await page.setViewportSize({width,height:900});
    await page.goto(info.base+info.path);
    await page.getByRole('tab',{name:'Checks'}).click();
    const geom=await page.evaluate(()=>({content:document.documentElement.scrollWidth,viewport:innerWidth}));
    check(`Checks at ${String(width)}px has no horizontal overflow`,geom.content<=geom.viewport+1,JSON.stringify(geom));
    await page.screenshot({path:`${shots}/checks-${String(width)}.png`});
  }
  check('no runtime exceptions in wizard and Checks',errors.length===0,errors.slice(0,3).join(' | '));
} finally {
  await browser.close();
}
console.log(failures===0?'ALL PASSED':`${failures} FAILED`);
process.exitCode=failures?1:0;

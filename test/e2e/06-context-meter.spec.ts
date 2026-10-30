// Actual browser and task API after full, compacted, and unknown usage states.
import { readFileSync, existsSync } from 'node:fs';
import { createRequire } from 'node:module';
import { execFileSync } from 'node:child_process';
const { chromium } = createRequire(import.meta.url)(process.env.PW as string) as typeof import('../../web/node_modules/playwright');
const info=JSON.parse(readFileSync(process.argv[2] as string,'utf8')) as {base:string;email:string;password:string;project:string;task:string;db:string};
function seed(used:number,window:number,state:'unknown'|'measured'|'compacting'|'failed'='measured'):void {
  execFileSync('python3',['-c',`import sqlite3,sys; c=sqlite3.connect(sys.argv[1]); c.execute("UPDATE tasks SET context_used=?, context_window=?, context_state=? WHERE id=?",(int(sys.argv[3]),int(sys.argv[4]),sys.argv[5],sys.argv[2])); c.commit(); c.close()`,info.db,info.task,String(used),String(window),state]);
}
const executablePath = process.env.CHROMIUM_BIN || (existsSync('/usr/bin/chromium-headless-shell') ? '/usr/bin/chromium-headless-shell' : undefined);
const browser=await chromium.launch({...(executablePath ? { executablePath } : {}),args:['--no-sandbox']});
try {
 const page=await browser.newPage({viewport:{width:1280,height:900}});
 await page.goto(info.base+'/login');
 await page.getByLabel(/email/i).fill(info.email);
 await page.locator('#password').fill(info.password);
 await page.getByRole('button',{name:/sign in/i}).click();
 await page.waitForURL(/\/app/,{timeout:15000});
 const path=`${info.base}/app/projects/${info.project}/tasks/${info.task}`;
 seed(31000,32000);
 await page.goto(path);
 await page.getByText('31K / 32K · 97%',{exact:true}).waitFor({timeout:15000});
 console.log('PASS browser shows full measured context');
 seed(6000,32000);
 await page.reload();
 await page.getByText('6K / 32K · 19%',{exact:true}).waitFor({timeout:15000});
 if(await page.getByText('31K / 32K · 97%',{exact:true}).count())throw Error('stale full meter');
 console.log('PASS browser shows reduced context after compact');
 seed(31000,32000,'compacting');
 await page.reload();
 await page.getByText('Compacting context…',{exact:true}).first().waitFor({timeout:15000});
 console.log('PASS browser shows active compaction instead of stale red');
 seed(7300,32000,'measured');
 await page.reload();
 await page.getByText('7K / 32K · 23%',{exact:true}).waitFor({timeout:15000});
 if(await page.getByText('Compacting context…',{exact:true}).count())throw Error('compacting label remained after successful compact');
 console.log('PASS browser returns to latest measured context after compaction');
 seed(31000,32000,'failed');
 await page.reload();
 await page.getByText('Context compaction failed',{exact:true}).first().waitFor({timeout:15000});
 console.log('PASS browser shows failed compaction without falsely resetting context');
 seed(0,0,'unknown');
 await page.reload();
 await page.locator('main header').waitFor();
 if(await page.locator('main header progress[aria-label="Context window"]').count())throw Error('unknown window renders a false percent');
 console.log('PASS browser hides unknown window');
}finally{await browser.close()}

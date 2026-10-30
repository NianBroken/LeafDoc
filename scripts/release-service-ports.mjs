// Source-tree tools delegate listener ownership to the shared Go implementation.
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import path from 'node:path';
const root=fileURLToPath(new URL('../',import.meta.url));
export function releaseFixedServicePorts(services) {
 const result=spawnSync('go',['run','./cmd/service-ports',...services],{cwd:root,stdio:'inherit',windowsHide:true});
 if(result.error) throw result.error;
 if(result.status!==0) throw new Error('Could not release fixed service ports');
}
if(process.argv[1] && path.resolve(process.argv[1])===fileURLToPath(import.meta.url)) releaseFixedServicePorts(process.argv.slice(2));

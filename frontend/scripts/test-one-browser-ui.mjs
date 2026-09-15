import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';

const eventSource = await readFile(new URL('../public/one-browser/proxy-node-events.js', import.meta.url), 'utf8');
const appSource = await readFile(new URL('../public/one-browser/app.js', import.meta.url), 'utf8');
const htmlSource = await readFile(new URL('../public/one-browser/index.html', import.meta.url), 'utf8');
const stage3Source = await readFile(new URL('../public/one-browser/stage3-fixes.css', import.meta.url), 'utf8');

const context = { module: { exports: {} }, globalThis: {} };
vm.runInNewContext(eventSource, context);
const { bind, singleFlightDelete } = context.module.exports;

let installedListener;
let listenerCount = 0;
const container = {
  addEventListener(type, listener) {
    assert.equal(type, 'click');
    listenerCount += 1;
    installedListener = listener;
  },
  removeEventListener() {},
};
let calls = 0;
let received;
let releaseDelete;
const pendingDelete = new Promise(resolve => { releaseDelete = resolve; });
const dispatch = singleFlightDelete((id, action) => {
  calls += 1;
  received = { id, action };
  return pendingDelete;
});
bind(container, dispatch);
assert.equal(listenerCount, 1, '代理容器只能注册一个点击监听器');

const card = { dataset: { proxyId: 'node-1' } };
const button = { dataset: { proxyId: 'node-1', proxyAction: 'delete' } };
let prevented = 0;
let stopped = 0;
const click = () => installedListener({
  target: { closest: selector => selector === '[data-proxy-action]' ? button : card },
  preventDefault: () => { prevented += 1; },
  stopPropagation: () => { stopped += 1; },
});
click();
click();
assert.equal(calls, 1, '一次删除点击只能分发一次');
assert.deepEqual(received, { id: 'node-1', action: 'delete' });
assert.equal(prevented, 2);
assert.equal(stopped, 2);
releaseDelete();
await pendingDelete;

assert.equal((appSource.match(/OneBrowserProxyNodeEvents\.bind\(/g) || []).length, 1, '应用只能绑定一次代理删除事件');
assert.equal((appSource.match(/singleFlightDelete\(/g) || []).length, 1, '删除操作必须使用单次执行封装');
assert.equal(appSource.includes('card.onclick='), false, '不应残留卡片级重复监听');
assert.equal(appSource.includes('deletingProxyNodeIds.has(id)'), true, '删除操作必须有单次执行保护');
assert.equal(htmlSource.includes('data-kernel-card="edge"'), false, '不应显示 Edge 内核');
assert.equal(htmlSource.includes('data-kernel-card="firefox"'), false, '不应显示 Firefox 内核');
assert.equal((htmlSource.match(/data-kernel-card=/g) || []).length, 1, '只能显示一个 fingerprint-chromium 内核');
assert.equal(htmlSource.includes('id="openChromeWebStore"'), true, '应用商店必须由窗口内打开处理');
assert.equal(htmlSource.includes('target="_blank"'), false, '应用商店不能再打开独立系统窗口');
assert.equal(htmlSource.includes('id="exportSystemLogs"'), true, '日志页必须提供一键导出');
assert.equal(htmlSource.includes('id="clearOldSystemLogs"'), true, '日志页必须提供30天以上清理');
assert.equal(appSource.includes("const noteDefaults=['暂无备注']"), true, '新窗口不能再默认显示德国客服主窗口');
assert.equal(appSource.includes('enabledPluginKeys()'), true, '启动窗口时必须同步启用的内置扩展');
assert.equal(appSource.includes("title=\"删除节点\"") && appSource.includes('>×</button>'), true, '代理删除按钮必须显示为 X');
assert.equal(appSource.includes('data-show-account-password'), false, '账号列表不能提供密码预览按钮');
assert.equal(htmlSource.includes('class="help-search"'), false, '帮助中心右侧搜索控件必须移除');
assert.equal(htmlSource.includes('<div class="help-version">当前版本 <span>1.8.3</span></div>'), true, '帮助中心必须显示当前版本');
assert.equal(stage3Source.includes('"PingFang SC"'), true, '界面字体必须优先使用苹方');
assert.equal(stage3Source.includes('box-shadow: none !important'), true, '筛选控件不能保留描边阴影');

const shellSource = await readFile(new URL('../src/OneBrowserShell.tsx', import.meta.url), 'utf8');
const mainSource = await readFile(new URL('../../main.go', import.meta.url), 'utf8');
const viewportSource = await readFile(new URL('../public/one-browser/viewport-fix.css', import.meta.url), 'utf8');
assert.equal(shellSource.includes('"app:request-close"'), true, '关闭事件必须由当前壳层处理');
assert.equal(shellSource.includes('await call("ForceQuit")'), true, '关闭事件必须真正退出应用');
assert.equal(mainSource.includes('WebviewGpuIsDisabled:                false'), true, 'Windows WebView GPU 必须启用');
assert.equal(viewportSource.includes('body.resize-active'), true, '调整窗口尺寸时必须降低毛玻璃重绘成本');
assert.equal(shellSource.includes('openChromeWebStore'), true, 'Chrome 应用商店必须在受管浏览器中打开');
assert.equal(shellSource.includes('connectGoogleDrive'), true, 'Google Drive 必须在受管浏览器中打开');

console.log('One Browser UI simulated click checks passed');

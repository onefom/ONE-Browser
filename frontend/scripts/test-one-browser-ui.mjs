import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import vm from 'node:vm';

const eventSource = await readFile(new URL('../public/one-browser/proxy-node-events.js', import.meta.url), 'utf8');
const appSource = await readFile(new URL('../public/one-browser/app.js', import.meta.url), 'utf8');
const htmlSource = await readFile(new URL('../public/one-browser/index.html', import.meta.url), 'utf8');
const stage3Source = await readFile(new URL('../public/one-browser/stage3-fixes.css', import.meta.url), 'utf8');
const stage4Source = await readFile(new URL('../public/one-browser/stage4-portable-data.css', import.meta.url), 'utf8');
const stage5Source = await readFile(new URL('../public/one-browser/stage5-window-polish.css', import.meta.url), 'utf8');
const stage6Source = await readFile(new URL('../public/one-browser/stage6-system-polish.css', import.meta.url), 'utf8');
const stage7Source = await readFile(new URL('../public/one-browser/stage7-unified-layout.css', import.meta.url), 'utf8');
const transferSource = await readFile(new URL('../../backend/internal/browser/download_core_transfer.go', import.meta.url), 'utf8');

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
assert.equal(appSource.includes('await window.oneBrowserPluginSync'), false, '启动窗口不能等待扩展网络下载');
assert.equal(appSource.includes('data-action="config"') && appSource.includes('data-action="network"'), true, '配置和网络图标必须提供点击操作');
assert.equal(appSource.includes('class="hover-card"'), false, '表格配置按钮不能残留会被裁切的悬浮黑条');
assert.equal(appSource.includes("document.querySelectorAll('.home-only-action').forEach"), true, '升级与云备份入口只能显示在窗口管理首页');
const sidebarOrder=['environments','kernels','proxy','accounts','plugins','logs','settings'];
const sidebarPositions=sidebarOrder.map(page=>htmlSource.indexOf(`data-page="${page}"`));
assert.equal(sidebarPositions.every((position,index)=>index===0||position>sidebarPositions[index-1]), true, '左侧菜单顺序必须符合产品要求');
assert.equal(appSource.includes("title=\"删除节点\"") && appSource.includes('>×</button>'), true, '代理删除按钮必须显示为 X');
assert.equal(appSource.includes('data-show-account-password'), false, '账号列表不能提供密码预览按钮');
assert.equal(htmlSource.includes('class="help-search"'), false, '帮助中心右侧搜索控件必须移除');
assert.equal(htmlSource.includes('<div class="help-version">当前版本 <span>1.8.10</span></div>'), true, '帮助中心必须显示当前版本');
assert.equal(stage3Source.includes('"PingFang SC"'), true, '界面字体必须优先使用苹方');
assert.equal(stage3Source.includes('box-shadow: none !important'), true, '筛选控件不能保留描边阴影');
assert.equal(htmlSource.includes('id="createFormScroll"'), true, '新建窗口必须使用独立滚动内容区');
assert.equal(htmlSource.includes('向下滑动查看更多'), true, '新建窗口必须显示下滑提示');
assert.equal(stage4Source.includes('scrollbar-width: none'), true, '新建窗口滚动条必须隐藏');
assert.equal(stage4Source.includes('.create-modal-actions'), true, '取消和创建窗口按钮必须固定在滚动区外');
assert.equal(appSource.includes("createScrollHint.classList.toggle('is-hidden'"), true, '滚到底部后必须隐藏下滑提示');
assert.equal(stage4Source.includes('flex-direction: column'), true, '下滑箭头和提示文字必须上下居中排列');
assert.equal(stage4Source.includes('rgba(235,238,255,.99)'), true, '下滑遮罩必须具有清晰可见的浅紫背景');
assert.equal(stage5Source.includes('gap: 8px'), true, '下滑箭头和文字之间必须保留清晰间距');
assert.equal(htmlSource.includes('id="configDetailModal"'), true, '查看配置必须打开二级弹窗');
assert.equal(appSource.includes("configDetailModal.classList.add('open')"), true, '查看配置不能只显示提示气泡');
assert.equal(stage5Source.includes('table-layout: fixed'), true, '启动按钮状态变化不能触发表格重排');
assert.equal(stage5Source.includes('width: 72px'), true, '启动和启动中按钮必须使用相同宽度');
assert.equal(appSource.includes("if(!installed){coreUpdateBadge.textContent='当前无内核'"), true, '未安装内核时升级入口必须显示无内核状态');
assert.equal(appSource.includes('setCoreUpdateNotice(coreUpdateAvailable&&!ignored)'), true, '只有检测到真实版本差异时才能显示升级红点');
assert.equal(appSource.includes("showToast('内核升级任务已准备，下次启动生效')"), false, '不能再显示没有实际升级依据的完成提示');
assert.equal(htmlSource.includes('Fingerprint-Chromium（推荐）'), true, '推荐 User Agent 文案必须统一');
assert.equal(htmlSource.includes('id="customUserAgentInput"'), true, '自定义 User Agent 必须提供输入区');
assert.equal(appSource.includes("uaMode==='custom'&&!customUA"), true, '自定义 User Agent 不能为空');
assert.equal(stage6Source.includes('border-top:0!important'), true, '下滑遮罩顶部不能出现硬分割线');
assert.equal(htmlSource.includes('id="importKernelDirectory"'), true, '内核页必须支持导入本地目录');
assert.equal(htmlSource.includes('data-log-level="DEBUG"'), true, '日志页必须提供级别筛选');
assert.equal(htmlSource.includes('id="initializeSystem"'), true, '系统设置必须提供初始化入口');
assert.equal(appSource.includes('currentLogLevel'), true, '日志级别筛选必须实际参与渲染');
assert.equal(stage7Source.includes('.table-wrap{margin-inline:0!important'), true, '上下内容区必须使用一致的左右边界');
assert.equal(htmlSource.includes('<h2>账号保险箱</h2>'), false, '账号页不能重复显示二级标题');
assert.equal(htmlSource.includes('<h2>代理节点</h2>'), false, '代理页不能重复显示二级标题');
assert.equal(htmlSource.includes('<h2>插件中心</h2>'), false, '插件页不能重复显示二级标题');
assert.equal(appSource.includes("roundedSelectPopover.className='rounded-select-popover'"), true, '下拉选项必须使用统一圆角浮层');
assert.equal(transferSource.includes('分段连接不稳定，已切换断点续传模式'), true, '内核下载必须在分段失败后回退续传模式');

const shellSource = await readFile(new URL('../src/OneBrowserShell.tsx', import.meta.url), 'utf8');
const mainSource = await readFile(new URL('../../main.go', import.meta.url), 'utf8');
const viewportSource = await readFile(new URL('../public/one-browser/viewport-fix.css', import.meta.url), 'utf8');
assert.equal(shellSource.includes('windowPosition: String(config.position || "左上")'), true, '启动请求必须传递左上窗口位置');
assert.equal(shellSource.includes('"app:request-close"'), true, '关闭事件必须由当前壳层处理');
assert.equal(shellSource.includes('await call("ForceQuit")'), true, '关闭事件必须真正退出应用');
assert.equal(mainSource.includes('WebviewGpuIsDisabled:                false'), true, 'Windows WebView GPU 必须启用');
assert.equal(mainSource.includes('WebviewUserDataPath:                 webviewData.Target'), true, 'WebView2 数据必须保存到便携 data 目录');
assert.equal(viewportSource.includes('body.resize-active'), true, '调整窗口尺寸时必须降低毛玻璃重绘成本');
assert.equal(shellSource.includes('openChromeWebStore'), true, 'Chrome 应用商店必须在受管浏览器中打开');
assert.equal(shellSource.includes('connectGoogleDrive'), true, 'Google Drive 必须在受管浏览器中打开');

console.log('One Browser UI simulated click checks passed');

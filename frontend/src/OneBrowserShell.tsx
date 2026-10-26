import { useEffect } from "react";

declare const __ONE_BROWSER_UI_DOCUMENT__: string;

const profileStoreKey = "one-browser-window-profiles-v2";

async function call(name: string, ...args: unknown[]) {
  const fn = (window as any).go?.main?.App?.[name];
  if (typeof fn !== "function") throw new Error(`桌面服务未就绪：${name}`);
  try { return await fn(...args); }
  catch (error) { throw error instanceof Error ? error : new Error(String(error)); }
}

function profiles(): Record<string, string> {
  try { return JSON.parse(localStorage.getItem(profileStoreKey) || "{}"); } catch { return {}; }
}

async function openInManagedBrowser(url: string) {
  const ids = [...new Set(Object.values(profiles()).filter(Boolean))];
  let lastError: unknown = new Error("请先启动一个浏览器窗口");
  for (const profileID of ids) {
    try { await call("OneBrowserOpenManagedURL", profileID, url); return true; } catch (error) { lastError = error; }
  }
  throw lastError;
}

async function waitForKernelTask(method: string, ...args: unknown[]) {
  const runtime = (window as any).runtime;
  await new Promise<void>((resolve, reject) => {
    let off: (() => void) | undefined;
    let finished = false;
    const finish = (error?: Error) => {
      if (finished) return;
      finished = true;
      if (off) off();
      error ? reject(error) : resolve();
    };
    if (runtime?.EventsOn) off = runtime.EventsOn("download:progress", (event: any) => {
      if (event?.phase === "done") finish();
      if (event?.phase === "error") finish(new Error(event?.message || "内核下载失败"));
    });
    call(method, ...args).catch((error: Error) => finish(error));
    if (!runtime?.EventsOn) finish(new Error("下载进度服务未就绪"));
  });
}

function browserStartRequest(input: any) {
 const config=input?.config||{},windowID=String(input?.id||" ").trim();
 return {
        coreId: String(config.coreId || ""), windowId: windowID, profileId: String(input?.profileId || ""), name: String(input.name || "One Browser 窗口"),
        proxyId: String(input?.proxy?.id || "__system__"), proxyName: String(input?.proxy?.name || ""),
        account: String(input?.account || ""), os: String(config.osName || config.os || "Windows 11"),
        language: String(config.language || ""), timezone: String(config.timezone || ""),
        userAgent: String(config.ua || ""), windowSize: String(config.size || ""),
        windowPosition: String(config.position || "左上"),
        extensions: Array.isArray(input?.extensions) ? input.extensions : [],
        extensionIds: Array.isArray(input?.extensionIds) ? input.extensionIds : [],
      };
}

let foregroundWindowStarts = 0;
let activeKernelFamily = "chrome";
let kernelTransferBusy = false;
async function downloadFamilyKernel(key: string) {
 if (kernelTransferBusy) throw new Error("正在下载内核，请等待完成");
 kernelTransferBusy=true; activeKernelFamily=key;
 try { await waitForKernelTask("OneBrowserDownloadFamilyKernel", key); return await call("OneBrowserFamilyKernelStatus", key); }
 finally { kernelTransferBusy=false; }
}


function installBridge() {
  if ((window as any).onlineDesktop) return;
  const runtime = (window as any).runtime;
  (window as any).onlineDesktop = {
    registerBrowserWindows: (entries: any[]) => {
      const saved: Record<string,string> = {};
      const used = new Set<string>();
      for (const entry of entries) {
        if (entry?.windowId && entry?.profileId && !used.has(entry.profileId)) {
          saved[String(entry.windowId)] = String(entry.profileId);
          used.add(entry.profileId);
        }
      }
      localStorage.setItem(profileStoreKey, JSON.stringify(saved));
    },
    listDetachedWindows: (ids: string[]) => call("OneBrowserDetachedWindows", ids),
    deleteBrowserWindow: (input: any) => call("OneBrowserDeleteWindow", String(input.id || ""), String(input.profileId || "")),
    renameBrowserWindow: (input: any) => call("OneBrowserRenameWindow", String(input.id || ""), String(input.profileId || ""), String(input.name || "")),
    syncWindowDirectories: async (entries: any[]) => {
      const warnings: string[] = [];
      for (const entry of [...entries]) {
        if (!entry?.windowId || !entry?.profileId) continue;
        await new Promise(resolve => setTimeout(resolve, 80));
        while (foregroundWindowStarts > 0) await new Promise(resolve => setTimeout(resolve, 50));
        try { await call("OneBrowserRenameWindow", entry.windowId, entry.profileId, entry.name); }
        catch (error) { warnings.push(`${entry.name}: ${error instanceof Error ? error.message : String(error)}`); }
      }
      return warnings;
    },
    getClientVersion: () => call("OneBrowserClientVersion"),
    checkClientUpdate: () => call("OneBrowserCheckClientUpdate"),
    downloadClientUpdate: () => call("OneBrowserDownloadClientUpdate"),
    getKernelStatus: async () => ({ chrome: await call("OneBrowserFamilyKernelStatus", "chrome"), camoufox: await call("OneBrowserFamilyKernelStatus", "camoufox") }),
    getLatestKernelRelease: (key = "chrome") => call("OneBrowserCheckFamilyKernelUpdate", key),
    listKernels: () => call("OneBrowserKernels"),
    getWindowKernelBindings: () => call("OneBrowserWindowKernelBindings"),
    setWindowKernel: (input: any) => call("OneBrowserSetWindowKernel", String(input.id || ""), String(input.profileId || ""), String(input.coreId || "")),
    deleteImportedPlugin: (id: string) => call("OneBrowserDeleteImportedPlugin", id),
    downloadKernel: downloadFamilyKernel,
    updateKernel: downloadFamilyKernel,
    importKernelArchive: async () => { const core: any = await call("BrowserCoreImportLocal"); return core ? {installed:true,coreId:core.coreId} : null; },
    importKernelDirectory: async () => { const core: any = await call("BrowserCoreImportLocalDirectory"); return core ? {installed:true,coreId:core.coreId} : null; },
    importClashSubscription: async (url: string) => (await call("OneBrowserImportClash", url) as any[]).map(node => ({ id: node.proxyId, name: node.proxyName, type: "clash", server: "Mihomo", port: "", group: node.groupName || "订阅节点" })),
    parseProxyText: async (text: string) => (await call("OneBrowserParseClashText", text) as any[]).map(node => ({ id: node.proxyId, name: node.proxyName, type: "clash", server: "Mihomo", port: "", group: node.groupName || "手动导入" })),
    deleteProxyNode: (id: string) => call("OneBrowserDeleteProxy", id),
    testProxyNodes: (ids: string[]) => call("OneBrowserProxyBatchTestSpeed", ids, 10),
    onProxySpeedResult: (callback: (event: any) => void) => runtime?.EventsOn?.("proxy:speed:result", callback),
    startIPMonitor: () => call("OneBrowserStartIPMonitor"),
    getIPChange: () => call("OneBrowserPendingIPChange"),
    resolveIPChange: (id: string, close: boolean) => call("OneBrowserResolveIPChange", id, close),
    onIPChange: (callback: (event: any) => void) => runtime?.EventsOn?.("one-browser:ip-change", callback),
    getWindowNetworkInfo: (id: string | number) => call("OneBrowserGetNetworkInfo", profiles()[String(id)] || ""),
    removeKernel: async (_key: string, coreId: string, windows: any[]) => {
      await call("OneBrowserDeleteKernel", coreId, windows);
    },
    startBrowserWindow: async (input: any) => {
      if (input?.browserKey !== "chrome") throw new Error("请选择 Chrome 或 Firefox 内核");
      const windowID = String(input?.id || "");
      if (!windowID || /^\d+$/.test(windowID)) throw new Error("窗口独立标识无效，请重新打开窗口管理页");
      foregroundWindowStarts++;
      try {
      const result: any = await call("OneBrowserStart", browserStartRequest(input));
      const saved = profiles();
      saved[windowID] = result.profileId;
      localStorage.setItem(profileStoreKey, JSON.stringify(saved));
      return { ok: result.running, profileId: result.profileId, warning: result.warning, message: result.running ? "" : "浏览器未能进入运行状态" };
      } finally { foregroundWindowStarts--; }
    },
    stopBrowserWindow: (id: string | number) => { const value = profiles()[String(id)]; return value ? call("OneBrowserStop", value) : Promise.resolve(); },
    onKernelProgress: (callback: (event: any) => void) => runtime?.EventsOn?.("download:progress", (event: any) => {
      const phase = String(event?.phase || "downloading");
      callback({
        key: activeKernelFamily,
        stage: phase === "done" ? "complete" : phase,
        percent: Number(event?.progress || 0),
        message: String(event?.message || (phase === "extracting" ? "正在解压内核…" : "正在下载内核…")),
      });
    }),
    onBrowserClosed: (callback: (event: any) => void) => runtime?.EventsOn?.("browser:instance:stopped", (profileID: string) => {
      const entry = Object.entries(profiles()).find(([, savedProfileID]) => savedProfileID === profileID);
      callback(entry ? entry[0] : profileID);
    }),
    onBrowserError: (callback: (event: any) => void) => runtime?.EventsOn?.("browser:instance:error", callback),
    openChromeWebStore: () => openInManagedBrowser("https://chromewebstore.google.com/"),
    connectGoogleDrive: () => openInManagedBrowser("https://drive.google.com/drive/my-drive"),
    setWindowPlugins: (input: any) => input.removedExtensionIds?.length ? call("OneBrowserSetWindowPluginsWithRemoval", String(input.id || ""), String(input.profileId || ""), input.keys || [], input.extensionIds || [], input.removedExtensionIds) : call("OneBrowserSetWindowPlugins", String(input.id || ""), String(input.profileId || ""), input.keys || [], input.extensionIds || []),
    restartBrowserWindow: (input: any) => call("OneBrowserRestartConfiguredWindow", browserStartRequest(input)),
    getBrowserWindowState: async (id: string) => { const items: any = await call("BrowserProfileList"); return Boolean(items.find((p: any) => p.profileId === profiles()[String(id)])?.running); },
    restartWindowForPlugins: (input: any) => call("OneBrowserRestartWindowForPlugins", String(input.id || ""), String(input.profileId || "")),
    importPlugin: async () => {
      try {
        const result: any = await call("OneBrowserImportWindowExtension");
        return { canceled: false, name: result?.name || "扩展程序", extensionId: result?.extensionId || "" };
      } catch (error) {
        if (String(error).includes("已取消选择")) return { canceled: true };
        throw error;
      }
    },
    saveLocalData: (snapshot: Record<string, unknown>) => call("OneBrowserSaveLocalData", snapshot),
    getSystemLogs: () => call("GetAppLogs"),
    clearSystemLogs: () => call("ClearAppLogs"),
    clearOldSystemLogs: (days = 30) => call("OneBrowserClearOldLogs", days),
    exportSystemLogs: () => call("OneBrowserExportLogs"),
    clearBrowserCache: () => call("OneBrowserClearCache"),
    exportAccounts: (accounts: any[]) => call("OneBrowserExportAccounts", accounts),
    importAccounts: () => call("OneBrowserImportAccounts"),
    exportSystemConfig: (snapshot: Record<string, unknown>) => call("OneBrowserExportConfiguration", snapshot),
    importSystemConfig: () => call("OneBrowserImportConfiguration"),
    initializeSystem: () => call("OneBrowserInitializeSystem"),
    logOperation: (level: string, method: string, success: boolean, message = "") => call("FrontendOperationLog", level, method, success, 0, message),
  };
}

export default function OneBrowserShell() {
  installBridge();
  useEffect(() => {
    document.title = "One Browser";
    const runtime = (window as any).runtime;
    const off = runtime?.EventsOn?.("app:request-close", () => {
      // Windows close keeps managed browser processes alive. The application
      // remains available from the tray, whose menu owns the explicit exits.
      runtime?.WindowHide?.();
    });
    return () => off?.();
  }, []);
  return <iframe title="One Browser" src={typeof __ONE_BROWSER_UI_DOCUMENT__ === "string" ? __ONE_BROWSER_UI_DOCUMENT__ : "/one-browser/index.html"} style={{ border: 0, display: "block", height: "100vh", width: "100vw" }} />;
}

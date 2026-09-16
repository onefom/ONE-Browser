import { useEffect } from "react";

const profileStoreKey = "one-browser-ant-profile-map";

function call(name: string, ...args: unknown[]) {
  const fn = (window as any).go?.main?.App?.[name];
  return typeof fn === "function" ? fn(...args) : Promise.reject(new Error(`桌面服务未就绪：${name}`));
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

function installBridge() {
  if ((window as any).onlineDesktop) return;
  const runtime = (window as any).runtime;
  (window as any).onlineDesktop = {
    getKernelStatus: async () => ({ chrome: await call("OneBrowserKernelStatus") }),
    downloadKernel: async (key: string) => {
      if (key !== "chrome") throw new Error("第一阶段只提供 fingerprint-chromium Windows x64 内核");
      await new Promise<void>((resolve, reject) => {
        let off: (() => void) | undefined;
        const finish = (error?: Error) => { if (off) off(); error ? reject(error) : resolve(); };
        if (runtime?.EventsOn) off = runtime.EventsOn("download:progress", (event: any) => {
          if (event?.phase === "done") finish();
          if (event?.phase === "error") finish(new Error(event?.message || "内核下载失败"));
        });
        call("OneBrowserDownloadFingerprintChromium").catch((error: Error) => finish(error));
        if (!runtime?.EventsOn) finish(new Error("下载进度服务未就绪"));
      });
      return call("OneBrowserKernelStatus");
    },
    importKernelArchive: async () => { await call("BrowserCoreImportLocal"); return call("OneBrowserKernelStatus"); },
    importKernelDirectory: async () => { await call("BrowserCoreImportLocalDirectory"); return call("OneBrowserKernelStatus"); },
    importClashSubscription: async (url: string) => (await call("OneBrowserImportClash", url) as any[]).map(node => ({ id: node.proxyId, name: node.proxyName, type: "clash", server: "Mihomo", port: "" })),
    parseProxyText: async (text: string) => (await call("OneBrowserParseClashText", text) as any[]).map(node => ({ id: node.proxyId, name: node.proxyName, type: "clash", server: "Mihomo", port: "" })),
    deleteProxyNode: (id: string) => call("OneBrowserDeleteProxy", id),
    removeKernel: async (key: string) => {
      if (key !== "chrome") throw new Error("仅支持 fingerprint-chromium Windows x64 内核");
      const state: any = await call("OneBrowserKernelStatus");
      if (state?.coreId) await call("BrowserCoreDelete", state.coreId);
    },
    startBrowserWindow: async (input: any) => {
      if (input?.browserKey !== "chrome") throw new Error("第一阶段仅支持 fingerprint-chromium");
      const saved = profiles();
      const config = input?.config || {};
      const result: any = await call("OneBrowserStart", {
        profileId: saved[String(input.id)] || "", name: String(input.name || "One Browser 窗口"),
        proxyId: String(input?.proxy?.id || "__system__"), proxyName: String(input?.proxy?.name || ""),
        account: String(input?.account || ""), os: String(config.osName || config.os || "Windows 11"),
        language: String(config.language || ""), timezone: String(config.timezone || ""),
        userAgent: String(config.ua || ""), windowSize: String(config.size || ""),
        windowPosition: String(config.position || "左上"),
        extensions: Array.isArray(input?.extensions) ? input.extensions : [],
      });
      saved[String(input.id)] = result.profileId;
      localStorage.setItem(profileStoreKey, JSON.stringify(saved));
      return { ok: result.running, message: result.running ? "" : "浏览器未能进入运行状态" };
    },
    stopBrowserWindow: (id: string | number) => { const value = profiles()[String(id)]; return value ? call("OneBrowserStop", value) : Promise.resolve(); },
    onKernelProgress: (callback: (event: any) => void) => runtime?.EventsOn?.("download:progress", (event: any) => {
      const phase = String(event?.phase || "downloading");
      callback({
        key: "chrome",
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
    importPlugin: async () => {
      const result: any = await call("BrowserExtensionInstallLocalFile");
      return { canceled: false, name: result?.name || "扩展程序" };
    },
    saveLocalData: (snapshot: Record<string, unknown>) => call("OneBrowserSaveLocalData", snapshot),
    syncBuiltInExtensions: (keys: string[]) => call("OneBrowserSyncBuiltinExtensions", keys),
    getSystemLogs: () => call("GetAppLogs"),
    clearSystemLogs: () => call("ClearAppLogs"),
    clearOldSystemLogs: (days = 30) => call("OneBrowserClearOldLogs", days),
    exportSystemLogs: () => call("OneBrowserExportLogs"),
    clearBrowserCache: () => call("OneBrowserClearCache"),
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
    let quitting = false;
    const off = runtime?.EventsOn?.("app:request-close", async () => {
      if (quitting) return;
      quitting = true;
      try {
        await call("ForceQuit");
      } catch (error) {
        console.error("关闭 One Browser 失败", error);
        quitting = false;
      }
    });
    return () => off?.();
  }, []);
  return <iframe title="One Browser" src="/one-browser/index.html" style={{ border: 0, display: "block", height: "100vh", width: "100vw" }} />;
}

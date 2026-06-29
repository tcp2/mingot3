// src/electron/preload.ts
var import_electron = require("electron");
var electronAPI = {
  /** Invoke an IPC handler in the main process */
  invoke: (channel, ...args) => {
    return import_electron.ipcRenderer.invoke(channel, ...args);
  },
  /** Subscribe to push events from the main process */
  on: (channel, listener) => {
    import_electron.ipcRenderer.on(channel, (_event, ...args) => listener(...args));
  },
  /** Unsubscribe from push events */
  off: (channel, _listener) => {
    import_electron.ipcRenderer.removeAllListeners(channel);
  }
};
import_electron.contextBridge.exposeInMainWorld("electronAPI", electronAPI);

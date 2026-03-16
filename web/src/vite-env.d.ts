/// <reference types="vite/client" />

declare module "@novnc/novnc" {
  const RFB: new (target: HTMLElement, url: string, options?: object) => {
    disconnect: () => void;
    addEventListener: (name: string, fn: () => void) => void;
    scaleViewport: boolean;
  };
  export default RFB;
}

interface ImportMetaEnv {
  readonly VITE_API_BASE?: string;
  readonly VITE_WS_BASE?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

import { System } from "@wailsio/runtime";

type NativeWebKitWindow = Window & {
  webkit?: { messageHandlers?: { external?: { postMessage?: unknown } } };
};

export function usesMacWindowChrome(): boolean {
  if (typeof window === "undefined" || !System.IsMac()) return false;
  // A server preview can report macOS too; only WKWebView has native controls to clear.
  return typeof (window as NativeWebKitWindow).webkit?.messageHandlers?.external?.postMessage === "function";
}

export {};
declare global {
  interface Window {
    Intercom: any;
    intercomSettings: any;
    attachEvent: any;
  }
  interface HTMLScriptElement {
    parentNode: any;
  }
}

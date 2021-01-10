declare module '*.pcss' {
    const content: {[className: string]: string};
    export default content;
}
declare module '*.css' {
    const content: {[className: string]: string};
    export default content;
}
declare module '*.png'
declare module '*.jpg'
declare let AUTH_TOKEN: string;
declare let MAIN_TOKEN: string | undefined;
declare let NO_CAPTCHA: boolean | undefined;
declare module 'dygraphs';
declare module '@novnc/novnc/core/rfb';
// cp - CloudPayments script
declare let cp: any;
declare const DEV: any;

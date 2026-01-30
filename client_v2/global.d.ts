declare module '*.svg' {
    const src: string;
    export default src;
}

declare module '*.module.pcss' {
    const classes: { [key: string]: string };
    export default classes;
}

declare module '*.pcss' {
    const classes: { [key: string]: string };
    export default classes;
}

declare module '*.module.pcss' {
  const classes: { [key: string]: string };
  export default classes;
}

declare module '*.pcss' {
  const classes: { [key: string]: string };
  export default classes;
}

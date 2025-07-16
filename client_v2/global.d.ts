import React from 'react';

declare module '*.svg' {
    const content: React.FunctionComponent<React.SVGAttributes<SVGElement>>;
    export default content;
}

declare module '*.module.pcss' {
    const classes: { [key: string]: string };
    export default classes;
}

declare module '*.pcss' {
    const classes: { [key: string]: string };
    export default classes;
}

import { Loader } from './Loader';

import s from './Loader.module.pcss';

export const PageLoader = () => (
    <Loader overlayClass={s.pageOverlay} class={s.pageLoader} icon="loader" />
);

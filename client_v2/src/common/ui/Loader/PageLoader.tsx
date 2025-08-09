import React from 'react';
import { Loader } from './Loader';

import s from './Loader.module.pcss';

export const PageLoader = () => <Loader overlayClassName={s.pageOverlay} className={s.pageLoader} icon="loader" />;

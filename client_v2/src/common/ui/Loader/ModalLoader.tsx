import React from 'react';
import { Loader } from './Loader';

import s from './Loader.module.pcss';

export const ModalLoader = () => <Loader overlayClassName={s.modalOverlay} className={s.modalLoader} icon="loader" />;

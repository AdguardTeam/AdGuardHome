import React from 'react';
import { Loader } from 'panel/common/ui';
import theme from 'panel/lib/theme';

export const CustomLoadingIndicator = () => (
    <Loader overlayClassName={theme.select.loaderOverlay} className={theme.select.loader} icon="loader" />
);

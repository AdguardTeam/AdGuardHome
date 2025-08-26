import React from 'react';

import intl from 'panel/common/intl';
import { StatusBlock } from './StatusBlock';

import s from './styles.module.pcss';

type Props = {
    type: 'warning' | 'error';
    message: string;
};

export const ValidationStatus = ({ type, message }: Props) => (
    <StatusBlock variant={type} title={intl.getMessage('encryption_certificate_has_issues')}>
        <div className={s.statusText}>{message}</div>
    </StatusBlock>
);

import React from 'react';
import intl from 'panel/common/intl';
import { StatusBlock } from './StatusBlock';

import s from './styles.module.pcss';

type Props = {
    validKey: boolean;
    keyType?: string;
};

export const KeyStatus = ({ validKey, keyType }: Props) => (
    <StatusBlock
        variant={validKey ? 'success' : 'error'}
        title={validKey ? intl.getMessage('encryption_key_valid') : intl.getMessage('encryption_key_invalid')}
    >
        {keyType && <div className={s.statusText}>{intl.getMessage('encryption_key_type', { value: keyType })}</div>}
    </StatusBlock>
);

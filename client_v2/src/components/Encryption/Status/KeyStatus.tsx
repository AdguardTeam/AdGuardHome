import React from 'react';
import intl from 'panel/common/intl';
import { StatusBlock } from './StatusBlock';

type Props = {
    validKey: boolean;
};

export const KeyStatus = ({ validKey }: Props) => (
    <StatusBlock
        variant={validKey ? 'success' : 'error'}
        title={validKey ? intl.getMessage('encryption_key_valid') : intl.getMessage('encryption_key_invalid')}
    />
);

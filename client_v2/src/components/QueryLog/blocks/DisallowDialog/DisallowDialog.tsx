import React from 'react';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';

import s from './DisallowDialog.module.pcss';

type Props = {
    ip: string;
    isAllowlistMode: boolean;
    onConfirm: () => void;
    onClose: () => void;
};

export const DisallowDialog = ({ ip, isAllowlistMode, onConfirm, onClose }: Props) => {
    return (
        <ConfirmDialog
            title={intl.getMessage('disallow_client_confirm_title')}
            text={
                <div className={s.body}>
                    <p>{intl.getMessage('disallow_client_confirm_text', { ip })}</p>
                    {isAllowlistMode && (
                        <p className={s.note}>
                            {intl.getMessage('disallow_client_confirm_allowlist_note', { ip })}
                        </p>
                    )}
                </div>
            }
            buttonText={intl.getMessage('yes')}
            cancelText={intl.getMessage('cancel')}
            buttonVariant="primary"
            onConfirm={onConfirm}
            onClose={onClose}
        />
    );
};

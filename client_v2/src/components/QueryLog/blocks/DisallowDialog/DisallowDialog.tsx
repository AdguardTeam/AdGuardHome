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
                <div>
                    <div>{intl.getMessage('disallow_client_confirm_text', { ip })}</div>
                    {isAllowlistMode && (
                        <div className={s.note}>
                            {intl.getMessage('disallow_client_confirm_allowlist_note', { ip })}
                        </div>
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

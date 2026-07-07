import { type JSX } from 'solid-js';
import cn from 'clsx';

import { Dialog } from 'panel/common/ui/Dialog';
import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';

import s from './ConfigDialog.module.pcss';

type Props = {
    open: boolean;
    title: string;
    onClose: () => void;
    onSubmit: () => void;
    processing?: boolean;
    submitDisabled?: boolean;
    class?: string;
    children?: JSX.Element;
    footer?: JSX.Element;
    description?: string;
};

export const ConfigDialog = (props: Props) => {
    const isDisabled = () => !!props.processing || !!props.submitDisabled;

    return (
        <Dialog
            visible={props.open}
            onClose={props.onClose}
            title={props.title}
            wrapClass={cn('rc-dialog-update', s.configDialog, props.class)}
        >
            {props.description && <div class={s.description}>{props.description}</div>}

            <fieldset disabled={!!props.processing} class={s.body}>
                {props.children}
            </fieldset>
            <div class={s.footer}>
                {props.footer}
                <Button
                    variant="primary"
                    class={s.saveButton}
                    disabled={isDisabled()}
                    data-testid="config-dialog-save"
                    onClick={props.onSubmit}
                >
                    {intl.getMessage('save')}
                </Button>
            </div>
        </Dialog>
    );
};

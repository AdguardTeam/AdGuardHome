import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';
import { Button } from 'panel/common/ui/Button';

import s from './StatusError.module.pcss';

type StatusErrorProps = {
    message: string;
    processing: boolean;
    onRetry: () => void;
};

export const StatusError = (props: StatusErrorProps) => {
    return (
        <div class={cn(theme.text.t2, s.root)} data-testid="status-error">
            <Icon icon="attention" class={s.icon} />
            <div class={s.message}>{props.message}</div>
            <Button
                variant="secondary"
                size="small"
                class={s.button}
                disabled={props.processing}
                onClick={props.onRetry}
            >
                {intl.getMessage('try_again')}
            </Button>
        </div>
    );
};

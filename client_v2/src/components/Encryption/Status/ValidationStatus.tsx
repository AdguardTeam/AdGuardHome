import intl from 'panel/common/intl';
import { StatusBlock } from './StatusBlock';

import s from './styles.module.pcss';

type Props = {
    type: 'warning' | 'error';
    message: string;
};

export const ValidationStatus = (props: Props) => (
    <StatusBlock variant={props.type} title={intl.getMessage('encryption_certificate_has_issues')}>
        <div class={s.statusText}>{props.message}</div>
    </StatusBlock>
);

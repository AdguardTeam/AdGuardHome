import { Show } from 'solid-js';
import { format } from 'date-fns';

import { EMPTY_DATE } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import { StatusBlock } from './StatusBlock';
import s from './styles.module.pcss';

type Props = {
    validChain: boolean;
    validCert: boolean;
    subject?: string;
    issuer?: string;
    notAfter?: string;
    dnsNames?: string[];
};

export const CertificateStatus = (props: Props) => (
    <StatusBlock
        variant={props.validChain ? 'success' : 'error'}
        title={
            props.validChain
                ? intl.getMessage('encryption_chain_valid')
                : intl.getMessage('encryption_chain_invalid')
        }
    >
        <Show
            when={
                props.validCert &&
                (props.subject || props.issuer || props.notAfter || props.dnsNames)
            }
        >
            <ul class={s.statusList}>
                <Show when={props.subject}>
                    <li>{intl.getMessage('encryption_subject', { value: props.subject })}</li>
                </Show>
                <Show when={props.issuer}>
                    <li>{intl.getMessage('encryption_issuer', { value: props.issuer })}</li>
                </Show>
                <Show when={props.notAfter && props.notAfter !== EMPTY_DATE}>
                    <li>
                        {intl.getMessage('encryption_expire', {
                            value: format(props.notAfter!, 'yyyy-MM-dd HH:mm:ss'),
                        })}
                    </li>
                </Show>
                <Show when={props.dnsNames}>
                    <li>
                        {intl.getMessage('encryption_hostnames', {
                            value: props.dnsNames!.join(', '),
                        })}
                    </li>
                </Show>
            </ul>
        </Show>
    </StatusBlock>
);

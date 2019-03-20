import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

const Examples = props => (
    <div className="list leading-loose">
        <Trans>examples_title</Trans>:
        <ol className="leading-loose">
            <li>
                <code>1.1.1.1</code> - { props.t('example_upstream_regular') }
            </li>
            <li>
                <code>tls://1dot1dot1dot1.cloudflare-dns.com</code> - <span dangerouslySetInnerHTML={{ __html: props.t('example_upstream_dot') }} />
            </li>
            <li>
                <code>https://cloudflare-dns.com/dns-query</code> - <span dangerouslySetInnerHTML={{ __html: props.t('example_upstream_doh') }} />
            </li>
            <li>
                <code>tcp://1.1.1.1</code> - { props.t('example_upstream_tcp') }
            </li>
            <li>
                <code>sdns://...</code> - <span dangerouslySetInnerHTML={{ __html: props.t('example_upstream_sdns') }} />
            </li>
            <li>
                <code>[/host.com/]1.1.1.1</code> - <span dangerouslySetInnerHTML={{ __html: props.t('example_upstream_reserved') }} />
            </li>
        </ol>
    </div>
);

Examples.propTypes = {
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Examples);

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import { Trans, withNamespaces } from 'react-i18next';
import Card from '../ui/Card';

class Upstream extends Component {
    handleChange = (e) => {
        const { value } = e.currentTarget;
        this.props.handleUpstreamChange(value);
    };

    handleSubmit = (e) => {
        e.preventDefault();
        this.props.handleUpstreamSubmit();
    };

    handleTest = () => {
        this.props.handleUpstreamTest();
    }

    render() {
        const testButtonClass = classnames({
            'btn btn-primary btn-standart mr-2': true,
            'btn btn-primary btn-standart mr-2 btn-loading': this.props.processingTestUpstream,
        });
        const { t } = this.props;

        return (
            <Card
                title={ t('Upstream DNS servers') }
                subtitle={ t('If you keep this field empty, AdGuard Home will use <a href="https://1.1.1.1/" target="_blank">Cloudflare DNS</a> as an upstream. Use tls:// prefix for DNS over TLS servers.') }
                bodyType="card-body box-body--settings"
            >
                <div className="row">
                    <div className="col">
                        <form>
                            <textarea
                                className="form-control form-control--textarea"
                                value={this.props.upstreamDns}
                                onChange={this.handleChange}
                            />
                            <div className="card-actions">
                                <button
                                    className={testButtonClass}
                                    type="button"
                                    onClick={this.handleTest}
                                >
                                    <Trans>Test upstreams</Trans>
                                </button>
                                <button
                                    className="btn btn-success btn-standart"
                                    type="submit"
                                    onClick={this.handleSubmit}
                                >
                                    <Trans>Apply</Trans>
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </Card>
        );
    }
}

Upstream.propTypes = {
    upstreamDns: PropTypes.string,
    processingTestUpstream: PropTypes.bool,
    handleUpstreamChange: PropTypes.func,
    handleUpstreamSubmit: PropTypes.func,
    handleUpstreamTest: PropTypes.func,
    t: PropTypes.func,
};

export default withNamespaces()(Upstream);

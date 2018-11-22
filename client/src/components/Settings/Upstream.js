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
                title={ t('upstream_dns') }
                subtitle={ t('upstream_dns_hint') }
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
                                    <Trans>test_upstream_btn</Trans>
                                </button>
                                <button
                                    className="btn btn-success btn-standart"
                                    type="submit"
                                    onClick={this.handleSubmit}
                                >
                                    <Trans>apply_btn</Trans>
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

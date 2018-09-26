import React, { Component } from 'react';
import PropTypes from 'prop-types';
import classnames from 'classnames';
import Card from '../ui/Card';

export default class Upstream extends Component {
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

        return (
            <Card
                title="Upstream DNS servers"
                subtitle="If you keep this field empty, AdGuard will use <a href='https://1.1.1.1/' target='_blank'>Cloudflare DNS</a> as an upstream."
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
                                    Test upstreams
                                </button>
                                <button
                                    className="btn btn-success btn-standart"
                                    type="submit"
                                    onClick={this.handleSubmit}
                                >
                                    Apply
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
};

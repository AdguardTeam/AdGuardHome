import React, { Component } from 'react';
import PropTypes from 'prop-types';
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

    render() {
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
                                className="form-control"
                                value={this.props.upstream}
                                onChange={this.handleChange}
                            />
                            <div className="card-actions">
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
    upstream: PropTypes.string,
    handleUpstreamChange: PropTypes.func,
    handleUpstreamSubmit: PropTypes.func,
};

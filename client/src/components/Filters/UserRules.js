import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Card from '../ui/Card';

export default class UserRules extends Component {
    handleChange = (e) => {
        const { value } = e.currentTarget;
        this.props.handleRulesChange(value);
    };

    handleSubmit = (e) => {
        e.preventDefault();
        this.props.handleRulesSubmit();
    };

    render() {
        return (
            <Card
                title="Custom filtering rules"
                subtitle="Enter one rule on a line. You can use either adblock rules or hosts files syntax."
            >
                <form onSubmit={this.handleSubmit}>
                    <textarea className="form-control" value={this.props.userRules} onChange={this.handleChange} />
                    <div className="card-actions">
                        <button
                            className="btn btn-success btn-standart"
                            type="submit"
                            onClick={this.handleSubmit}
                        >
                            Apply...
                        </button>
                    </div>
                </form>
                <hr/>
                <div className="list leading-loose">
                    Examples:
                    <ol className="leading-loose">
                        <li>
                            <code>||example.org^</code> - block access to the example.org domain
                            and all its subdomains
                        </li>
                        <li>
                            <code> @@||example.org^</code> - unblock access to the example.org
                            domain and all its subdomains
                        </li>
                        <li>
                            <code>example.org 127.0.0.1</code> - AdGuard DNS will now return
                            127.0.0.1 address for the example.org domain (but not its subdomains).
                        </li>
                        <li>
                            <code>! Here goes a comment</code> - just a comment
                        </li>
                        <li>
                            <code># Also a comment</code> - just a comment
                        </li>
                    </ol>
                </div>
            </Card>
        );
    }
}

UserRules.propTypes = {
    userRules: PropTypes.string,
    handleRulesChange: PropTypes.func,
    handleRulesSubmit: PropTypes.func,
};

import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import Upstream from './Upstream';
import Checkbox from '../ui/Checkbox';
import Loading from '../ui/Loading';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import './Settings.css';

export default class Settings extends Component {
    settings = {
        filtering: {
            enabled: false,
            title: 'Block domains using filters and hosts files',
            subtitle: 'You can setup blocking rules in the <a href="#filters">Filters</a> settings.',
        },
        safebrowsing: {
            enabled: false,
            title: 'Use AdGuard browsing security web service',
            subtitle: 'AdGuard DNS will check if domain is blacklisted by the browsing security web service (sb.adtidy.org). It will use privacy-safe lookup API to do the check.',
        },
        parental: {
            enabled: false,
            title: 'Use AdGuard parental control web service',
            subtitle: 'AdGuard DNS will check if domain contains adult materials. It uses the same privacy-friendly API as the browsing security web service.',
        },
        safesearch: {
            enabled: false,
            title: 'Enforce safe search',
            subtitle: 'AdGuard DNS can enforce safe search in the major search engines: Google, Bing, Yandex.',
        },
    };

    componentDidMount() {
        this.props.initSettings(this.settings);
    }

    handleUpstreamChange = (value) => {
        this.props.handleUpstreamChange({ upstream: value });
    };

    handleUpstreamSubmit = () => {
        this.props.setUpstream(this.props.settings.upstream);
    };

    handleUpstreamTest = () => {
        this.props.testUpstream(this.props.settings.upstream);
    };

    renderSettings = (settings) => {
        if (Object.keys(settings).length > 0) {
            return Object.keys(settings).map((key) => {
                const setting = settings[key];
                const { enabled } = setting;
                return (<Checkbox
                    key={key}
                    {...settings[key]}
                    handleChange={() => this.props.toggleSetting(key, enabled)}
                    />);
            });
        }
        return (
            <div>No settings</div>
        );
    }

    render() {
        const { settings, upstream } = this.props;
        return (
            <Fragment>
                <PageTitle title="Settings" />
                {settings.processing && <Loading />}
                {!settings.processing &&
                    <div className="content">
                        <div className="row">
                            <div className="col-md-12">
                                <Card title="General settings" bodyType="card-body box-body--settings">
                                    <div className="form">
                                        {this.renderSettings(settings.settingsList)}
                                    </div>
                                </Card>
                                <Upstream
                                    upstream={upstream}
                                    processingTestUpstream={settings.processingTestUpstream}
                                    handleUpstreamChange={this.handleUpstreamChange}
                                    handleUpstreamSubmit={this.handleUpstreamSubmit}
                                    handleUpstreamTest={this.handleUpstreamTest}
                                />
                            </div>
                        </div>
                    </div>
                }
            </Fragment>
        );
    }
}

Settings.propTypes = {
    initSettings: PropTypes.func,
    settings: PropTypes.object,
    settingsList: PropTypes.object,
    toggleSetting: PropTypes.func,
    handleUpstreamChange: PropTypes.func,
    setUpstream: PropTypes.func,
    upstream: PropTypes.string,
};

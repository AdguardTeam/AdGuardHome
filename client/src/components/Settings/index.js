import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces, Trans } from 'react-i18next';
import Upstream from './Upstream';
import Checkbox from '../ui/Checkbox';
import Loading from '../ui/Loading';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import './Settings.css';

class Settings extends Component {
    settings = {
        filtering: {
            enabled: false,
            title: this.props.t('Block domains using filters and hosts files'),
            subtitle: this.props.t('You can setup blocking rules in the <a href="#filters">Filters</a> settings.'),
        },
        safebrowsing: {
            enabled: false,
            title: this.props.t('Use AdGuard browsing security web service'),
            subtitle: this.props.t('AdGuard Home will check if domain is blacklisted by the browsing security web service. It will use privacy-friendly lookup API to perform the check: only a short prefix of the domain name SHA256 hash is sent to the server.'),
        },
        parental: {
            enabled: false,
            title: this.props.t('Use AdGuard parental control web service'),
            subtitle: this.props.t('AdGuard Home will check if domain contains adult materials. It uses the same privacy-friendly API as the browsing security web service.'),
        },
        safesearch: {
            enabled: false,
            title: this.props.t('Enforce safe search'),
            subtitle: this.props.t('AdGuard Home can enforce safe search in the following search engines: Google, Bing, Yandex.'),
        },
    };

    componentDidMount() {
        this.props.initSettings(this.settings);
    }

    handleUpstreamChange = (value) => {
        this.props.handleUpstreamChange({ upstreamDns: value });
    };

    handleUpstreamSubmit = () => {
        this.props.setUpstream(this.props.dashboard.upstreamDns);
    };

    handleUpstreamTest = () => {
        if (this.props.dashboard.upstreamDns.length > 0) {
            this.props.testUpstream(this.props.dashboard.upstreamDns);
        } else {
            this.props.addErrorToast({ error: this.props.t('No servers specified') });
        }
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
            <div><Trans>No settings</Trans></div>
        );
    }

    render() {
        const { settings, t } = this.props;
        const { upstreamDns } = this.props.dashboard;
        return (
            <Fragment>
                <PageTitle title={ t('Settings') } />
                {settings.processing && <Loading />}
                {!settings.processing &&
                    <div className="content">
                        <div className="row">
                            <div className="col-md-12">
                                <Card title={ t('General settings') } bodyType="card-body box-body--settings">
                                    <div className="form">
                                        {this.renderSettings(settings.settingsList)}
                                    </div>
                                </Card>
                                <Upstream
                                    upstreamDns={upstreamDns}
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
    t: PropTypes.func,
};

export default withNamespaces()(Settings);

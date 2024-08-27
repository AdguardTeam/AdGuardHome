import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import PageTitle from '../ui/PageTitle';

import Loading from '../ui/Loading';
import './LicenseSettings.css';

interface LicenseInfo {
  key: string;
  expiration: string;
}

const LicenseSettings: React.FC = () => {
  const { t } = useTranslation();
  const [licenseInfo, setLicenseInfo] = useState<LicenseInfo | null>(null);
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    const fetchLicenseInfo = async () => {
      try {
        const response = await fetch('/api/license');
        const data: LicenseInfo = await response.json();
        setLicenseInfo(data);
      } catch (error) {
        console.error('ライセンス情報の取得に失敗しました', error);
      } finally {
        setLoading(false);
      }
    };

    fetchLicenseInfo();
  }, []);

  if (loading) {
    return <Loading />;
  }

  return (
    <>
      <PageTitle title={t('license_settings')} />
      <div className="settings-container">
        {licenseInfo ? (
          <div className="license-info">
            <div className="form-group">
              <label>{t('license_key')}:</label>
              <input type="text" value={licenseInfo.key} readOnly />
            </div>
            <div className="form-group">
              <label>{t('expiration_date')}:</label>
              <input type="text" value={new Date(licenseInfo.expiration).toLocaleDateString()} readOnly />
            </div>
            <div className="form-group">
              <label>{t('days_remaining')}:</label>
              <input
                type="text"
                value={`${Math.ceil(
                  (new Date(licenseInfo.expiration).getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24)
                )} ${t('days')}`}
                readOnly
              />
            </div>
          </div>
        ) : (
          <p>{t('license_not_found')}</p>
        )}
      </div>
    </>
  );
};

export default LicenseSettings;

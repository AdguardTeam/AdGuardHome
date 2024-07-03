export const getFullDayName = (t: any, abbreviation: any) => {
    const dayMap = {
        sun: t('sunday'),
        mon: t('monday'),
        tue: t('tuesday'),
        wed: t('wednesday'),
        thu: t('thursday'),
        fri: t('friday'),
        sat: t('saturday'),
    };

    return dayMap[abbreviation] || '';
};

export const getShortDayName = (t: any, abbreviation: any) => {
    const dayMap = {
        sun: t('sunday_short'),
        mon: t('monday_short'),
        tue: t('tuesday_short'),
        wed: t('wednesday_short'),
        thu: t('thursday_short'),
        fri: t('friday_short'),
        sat: t('saturday_short'),
    };

    return dayMap[abbreviation] || '';
};

export const getTimeFromMs = (value: any) => {
    const selectedTime = new Date(value);
    const hours = selectedTime.getUTCHours();
    const minutes = selectedTime.getUTCMinutes();

    return {
        hours: hours.toString().padStart(2, '0'),

        minutes: minutes.toString().padStart(2, '0'),
    };
};

export const convertTimeToMs = (hours: any, minutes: any) => {
    const selectedTime = new Date(0);
    selectedTime.setUTCHours(parseInt(hours, 10));
    selectedTime.setUTCMinutes(parseInt(minutes, 10));

    return selectedTime.getTime();
};

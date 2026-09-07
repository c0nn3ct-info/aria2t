import { isLocale, setLocale } from '../i18n';
import { mountPage } from '../main';
import { LandingPage } from '../pages/landing';

const lang = document.documentElement.lang;
setLocale(isLocale(lang) ? lang : 'en');
mountPage(<LandingPage />);

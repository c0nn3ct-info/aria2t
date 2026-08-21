import { isLocale, setLocale } from '../i18n';
import { mountPage } from '../main';
import { ExtensionPage } from '../pages/extension';

const lang = document.documentElement.lang;
setLocale(isLocale(lang) ? lang : 'en');
mountPage(<ExtensionPage />);

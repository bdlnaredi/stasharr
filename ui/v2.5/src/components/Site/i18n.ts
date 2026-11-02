// The site front end's words in English, Hindi, French, Russian and
// Spanish. Each visitor picks a language from the globe button in the header;
// the first visit follows the browser's language. Names, titles and the
// admin's own row headings are shown as they are.

export type Lang = "en" | "hi" | "fr" | "ru" | "es";

export const languages: { id: Lang; label: string }[] = [
  { id: "en", label: "English" },
  { id: "hi", label: "हिन्दी" },
  { id: "fr", label: "Français" },
  { id: "ru", label: "Русский" },
  { id: "es", label: "Español" },
];

const KEY = "stasharr-site-lang";

function detect(): Lang {
  try {
    const saved = localStorage.getItem(KEY) as Lang | null;
    if (saved && languages.some((l) => l.id === saved)) return saved;
  } catch {
    // private window: fall back to the browser's language
  }
  const nav = (navigator.languages?.[0] ?? navigator.language ?? "en").slice(0, 2).toLowerCase();
  return (languages.some((l) => l.id === nav) ? nav : "en") as Lang;
}

let current: Lang = detect();

export function getLang(): Lang {
  return current;
}

export function setLang(l: Lang) {
  current = l;
  try {
    localStorage.setItem(KEY, l);
  } catch {
    // not remembered in a private window
  }
}

/** The locale for dates and numbers. */
export function locale(): string {
  return { en: "en-GB", hi: "hi-IN", fr: "fr-FR", ru: "ru-RU", es: "es-ES" }[current];
}

type Tr = { hi: string; fr: string; ru: string; es: string };

const dict: Record<string, Tr> = {
  // header and footer
  Home: { hi: "होम", fr: "Accueil", ru: "Главная", es: "Inicio" },
  Scenes: { hi: "दृश्य", fr: "Scènes", ru: "Сцены", es: "Escenas" },
  Pornstars: { hi: "पोर्नस्टार", fr: "Pornstars", ru: "Порнозвёзды", es: "Pornstars" },
  Models: { hi: "मॉडल", fr: "Modèles", ru: "Модели", es: "Modelos" },
  Categories: { hi: "श्रेणियाँ", fr: "Catégories", ru: "Категории", es: "Categorías" },
  Channels: { hi: "चैनल", fr: "Chaînes", ru: "Каналы", es: "Canales" },
  Reels: { hi: "रील्स", fr: "Reels", ru: "Рилсы", es: "Reels" },
  "My list": { hi: "मेरी सूची", fr: "Ma liste", ru: "Мой список", es: "Mi lista" },
  Account: { hi: "खाता", fr: "Compte", ru: "Аккаунт", es: "Cuenta" },
  "Admin interface": { hi: "एडमिन इंटरफ़ेस", fr: "Interface d'administration", ru: "Панель администратора", es: "Panel de administración" },
  "Log out": { hi: "लॉग आउट", fr: "Se déconnecter", ru: "Выйти", es: "Cerrar sesión" },
  Menu: { hi: "मेनू", fr: "Menu", ru: "Меню", es: "Menú" },
  Search: { hi: "खोजें", fr: "Rechercher", ru: "Поиск", es: "Buscar" },
  "Search scenes and stars": {
    hi: "दृश्य और स्टार खोजें",
    fr: "Rechercher scènes et stars",
    ru: "Поиск сцен и звёзд",
    es: "Buscar escenas y estrellas",
  },
  Language: { hi: "भाषा", fr: "Langue", ru: "Язык", es: "Idioma" },
  "All models were 18 years of age or older at the time of filming.": {
    hi: "फ़िल्मांकन के समय सभी मॉडल 18 वर्ष या उससे अधिक आयु के थे।",
    fr: "Tous les modèles avaient 18 ans ou plus au moment du tournage.",
    ru: "Всем моделям на момент съёмок было 18 лет или больше.",
    es: "Todos los modelos tenían 18 años o más en el momento de la grabación.",
  },

  // age check
  "This is an adult website": {
    hi: "यह एक वयस्क वेबसाइट है",
    fr: "Ceci est un site pour adultes",
    ru: "Это сайт для взрослых",
    es: "Este es un sitio para adultos",
  },
  "It contains sexually explicit material. Only continue if you are at least 18 years old, or the age of majority where you live, and you want to see it.": {
    hi: "इसमें स्पष्ट यौन सामग्री है। आगे तभी बढ़ें जब आपकी आयु कम से कम 18 वर्ष हो, या आपके यहाँ की वयस्कता की आयु हो, और आप इसे देखना चाहते हों।",
    fr: "Il contient du contenu sexuellement explicite. Continuez uniquement si vous avez au moins 18 ans, ou l'âge de la majorité dans votre pays, et si vous souhaitez le voir.",
    ru: "Сайт содержит откровенные сексуальные материалы. Продолжайте, только если вам исполнилось 18 лет или вы достигли совершеннолетия в своей стране и хотите это видеть.",
    es: "Contiene material sexual explícito. Continúa solo si tienes al menos 18 años, o la mayoría de edad donde vives, y quieres verlo.",
  },
  "I am 18 or older — Enter": {
    hi: "मेरी आयु 18 या अधिक है — प्रवेश करें",
    fr: "J'ai 18 ans ou plus — Entrer",
    ru: "Мне 18 или больше — Войти",
    es: "Tengo 18 años o más — Entrar",
  },
  Leave: { hi: "बाहर जाएँ", fr: "Quitter", ru: "Уйти", es: "Salir" },
  "Previewing the {name} style.": {
    hi: "{name} शैली का पूर्वावलोकन।",
    fr: "Aperçu du style {name}.",
    ru: "Предпросмотр стиля {name}.",
    es: "Vista previa del estilo {name}.",
  },
  "Stop preview": { hi: "पूर्वावलोकन बंद करें", fr: "Arrêter l'aperçu", ru: "Закрыть предпросмотр", es: "Salir de la vista previa" },

  // cards and lists
  New: { hi: "नया", fr: "Nouveau", ru: "Новое", es: "Nuevo" },
  "View all": { hi: "सभी देखें", fr: "Tout voir", ru: "Смотреть все", es: "Ver todo" },
  Untitled: { hi: "बिना शीर्षक", fr: "Sans titre", ru: "Без названия", es: "Sin título" },
  Back: { hi: "पीछे", fr: "Retour", ru: "Назад", es: "Atrás" },
  More: { hi: "और", fr: "Plus", ru: "Ещё", es: "Más" },
  "‹ Prev": { hi: "‹ पिछला", fr: "‹ Préc.", ru: "‹ Назад", es: "‹ Anterior" },
  "Next ›": { hi: "अगला ›", fr: "Suiv. ›", ru: "Далее ›", es: "Siguiente ›" },

  // home
  Featured: { hi: "विशेष", fr: "À la une", ru: "Избранное", es: "Destacado" },
  Starring: { hi: "अभिनीत", fr: "Avec", ru: "В ролях", es: "Con" },
  "▶ Watch now": { hi: "▶ अभी देखें", fr: "▶ Regarder", ru: "▶ Смотреть", es: "▶ Ver ahora" },
  "+ My list": { hi: "+ मेरी सूची", fr: "+ Ma liste", ru: "+ Мой список", es: "+ Mi lista" },
  "✓ In my list": { hi: "✓ मेरी सूची में", fr: "✓ Dans ma liste", ru: "✓ В моём списке", es: "✓ En mi lista" },
  "Up next": { hi: "आगे", fr: "À suivre", ru: "Далее", es: "A continuación" },
  "Just added": { hi: "अभी जोड़ा गया", fr: "Nouveautés", ru: "Новинки", es: "Recién añadido" },
  "Explore ›": { hi: "देखें ›", fr: "Découvrir ›", ru: "Смотреть ›", es: "Explorar ›" },
  "Show {title}": { hi: "{title} दिखाएँ", fr: "Afficher {title}", ru: "Показать {title}", es: "Mostrar {title}" },
  "Continue watching": { hi: "देखना जारी रखें", fr: "Reprendre", ru: "Продолжить просмотр", es: "Seguir viendo" },
  "Latest updates": { hi: "नवीनतम", fr: "Dernières mises à jour", ru: "Последние обновления", es: "Últimas novedades" },
  "Top pornstars": { hi: "टॉप पोर्नस्टार", fr: "Top pornstars", ru: "Лучшие порнозвёзды", es: "Top pornstars" },
  "Top models": { hi: "टॉप मॉडल", fr: "Top modèles", ru: "Лучшие модели", es: "Top modelos" },
  "Trending now": { hi: "अभी ट्रेंडिंग", fr: "Tendances", ru: "Сейчас в тренде", es: "Tendencias" },
  "Most viewed": { hi: "सबसे ज़्यादा देखे गए", fr: "Les plus vues", ru: "Самые просматриваемые", es: "Más vistos" },
  "Picked for you": { hi: "आपके लिए चुने गए", fr: "Pour vous", ru: "Для вас", es: "Para ti" },
  "New from your favourites": {
    hi: "आपके पसंदीदा से नया",
    fr: "Nouveautés de vos favoris",
    ru: "Новое от ваших избранных",
    es: "Novedades de tus favoritas",
  },
  "Your liked scenes": { hi: "आपके पसंद किए दृश्य", fr: "Vos scènes aimées", ru: "Понравившиеся сцены", es: "Escenas que te gustan" },
  "Top rated": { hi: "सबसे ज़्यादा रेटेड", fr: "Les mieux notées", ru: "Лучшие по оценкам", es: "Mejor valorados" },

  // scene lists
  Latest: { hi: "नवीनतम", fr: "Récentes", ru: "Новые", es: "Recientes" },
  Trending: { hi: "ट्रेंडिंग", fr: "Tendances", ru: "В тренде", es: "Tendencias" },
  "Release date": { hi: "रिलीज़ तिथि", fr: "Date de sortie", ru: "Дата выхода", es: "Fecha de estreno" },
  Longest: { hi: "सबसे लंबे", fr: "Les plus longues", ru: "Самые длинные", es: "Más largos" },
  Random: { hi: "कोई भी", fr: "Au hasard", ru: "Случайные", es: "Aleatorio" },
  "Most scenes": { hi: "सबसे ज़्यादा दृश्य", fr: "Le plus de scènes", ru: "Больше всего сцен", es: "Más escenas" },
  Newest: { hi: "सबसे नए", fr: "Les plus récentes", ru: "Новые", es: "Más nuevas" },
  "A–Z": { hi: "A–Z", fr: "A–Z", ru: "А–Я", es: "A–Z" },
  Category: { hi: "श्रेणी", fr: "Catégorie", ru: "Категория", es: "Categoría" },
  Length: { hi: "अवधि", fr: "Durée", ru: "Длительность", es: "Duración" },
  Quality: { hi: "गुणवत्ता", fr: "Qualité", ru: "Качество", es: "Calidad" },
  Channel: { hi: "चैनल", fr: "Chaîne", ru: "Канал", es: "Canal" },
  Any: { hi: "कोई भी", fr: "Toutes", ru: "Любой", es: "Cualquiera" },
  "Under 10 min": { hi: "10 मिनट से कम", fr: "Moins de 10 min", ru: "До 10 мин", es: "Menos de 10 min" },
  "10–30 min": { hi: "10–30 मिनट", fr: "10–30 min", ru: "10–30 мин", es: "10–30 min" },
  "30–60 min": { hi: "30–60 मिनट", fr: "30–60 min", ru: "30–60 мин", es: "30–60 min" },
  "Over 1 hour": { hi: "1 घंटे से ज़्यादा", fr: "Plus d'1 heure", ru: "Больше часа", es: "Más de 1 hora" },
  "1080p and up": { hi: "1080p और ऊपर", fr: "1080p et plus", ru: "1080p и выше", es: "1080p o más" },
  "720p and up": { hi: "720p और ऊपर", fr: "720p et plus", ru: "720p и выше", es: "720p o más" },
  "Clear filters": { hi: "फ़िल्टर हटाएँ", fr: "Effacer les filtres", ru: "Сбросить фильтры", es: "Quitar filtros" },
  "All scenes": { hi: "सभी दृश्य", fr: "Toutes les scènes", ru: "Все сцены", es: "Todas las escenas" },
  "Results for": { hi: "परिणाम:", fr: "Résultats pour", ru: "Результаты по запросу", es: "Resultados para" },
  "No scenes match.": { hi: "कोई दृश्य नहीं मिला।", fr: "Aucune scène ne correspond.", ru: "Сцен не найдено.", es: "Ninguna escena coincide." },
  "No scenes here yet.": { hi: "यहाँ अभी कोई दृश्य नहीं है।", fr: "Aucune scène pour l'instant.", ru: "Здесь пока нет сцен.", es: "Aún no hay escenas." },

  // scene page
  "This scene is not available.": {
    hi: "यह दृश्य उपलब्ध नहीं है।",
    fr: "Cette scène n'est pas disponible.",
    ru: "Эта сцена недоступна.",
    es: "Esta escena no está disponible.",
  },
  "More of {name}": { hi: "{name} के और दृश्य", fr: "Plus de {name}", ru: "Ещё с {name}", es: "Más de {name}" },
  "You may also like": { hi: "आपको यह भी पसंद आ सकता है", fr: "Vous aimerez aussi", ru: "Вам также может понравиться", es: "También te puede gustar" },
  "Up next in {n}s": { hi: "{n} सेकंड में अगला", fr: "Suivante dans {n} s", ru: "Следующая через {n} с", es: "Siguiente en {n} s" },
  "▶ Play now": { hi: "▶ अभी चलाएँ", fr: "▶ Lire", ru: "▶ Смотреть сейчас", es: "▶ Reproducir" },
  Cancel: { hi: "रद्द करें", fr: "Annuler", ru: "Отмена", es: "Cancelar" },
  "▣ Normal view": { hi: "▣ सामान्य दृश्य", fr: "▣ Vue normale", ru: "▣ Обычный вид", es: "▣ Vista normal" },
  "▭ Theatre mode": { hi: "▭ थिएटर मोड", fr: "▭ Mode cinéma", ru: "▭ Режим кинотеатра", es: "▭ Modo cine" },
  Like: { hi: "पसंद", fr: "J'aime", ru: "Нравится", es: "Me gusta" },
  Liked: { hi: "पसंद किया", fr: "Aimé", ru: "Понравилось", es: "Te gusta" },
  Unlike: { hi: "नापसंद", fr: "Je n'aime plus", ru: "Убрать отметку", es: "Ya no me gusta" },
  "Your rating": { hi: "आपकी रेटिंग", fr: "Votre note", ru: "Ваша оценка", es: "Tu valoración" },
  "Rate it": { hi: "रेटिंग दें", fr: "Notez-la", ru: "Оцените", es: "Valórala" },

  // models
  "No models found.": { hi: "कोई मॉडल नहीं मिला।", fr: "Aucun modèle trouvé.", ru: "Модели не найдены.", es: "No se encontraron modelos." },
  All: { hi: "सभी", fr: "Tous", ru: "Все", es: "Todos" },
  "This model is not available.": {
    hi: "यह मॉडल उपलब्ध नहीं है।",
    fr: "Ce modèle n'est pas disponible.",
    ru: "Эта модель недоступна.",
    es: "Esta modelo no está disponible.",
  },
  Age: { hi: "आयु", fr: "Âge", ru: "Возраст", es: "Edad" },
  From: { hi: "देश", fr: "Origine", ru: "Откуда", es: "País" },
  Ethnicity: { hi: "जातीयता", fr: "Origine ethnique", ru: "Этническая принадлежность", es: "Etnia" },
  Hair: { hi: "बाल", fr: "Cheveux", ru: "Волосы", es: "Pelo" },
  Eyes: { hi: "आँखें", fr: "Yeux", ru: "Глаза", es: "Ojos" },
  Height: { hi: "ऊँचाई", fr: "Taille", ru: "Рост", es: "Altura" },
  Measurements: { hi: "माप", fr: "Mensurations", ru: "Параметры", es: "Medidas" },
  Career: { hi: "करियर", fr: "Carrière", ru: "Карьера", es: "Carrera" },
  now: { hi: "अब तक", fr: "aujourd'hui", ru: "наст. время", es: "actualidad" },
  Tattoos: { hi: "टैटू", fr: "Tatouages", ru: "Татуировки", es: "Tatuajes" },
  Piercings: { hi: "पियर्सिंग", fr: "Piercings", ru: "Пирсинг", es: "Piercings" },
  Pornstar: { hi: "पोर्नस्टार", fr: "Pornstar", ru: "Порнозвезда", es: "Pornstar" },
  Model: { hi: "मॉडल", fr: "Modèle", ru: "Модель", es: "Modelo" },
  "Also known as {names}": { hi: "अन्य नाम: {names}", fr: "Aussi connue sous le nom de {names}", ru: "Также известна как {names}", es: "También conocida como {names}" },
  "Most popular scenes": { hi: "सबसे लोकप्रिय दृश्य", fr: "Scènes les plus populaires", ru: "Самые популярные сцены", es: "Escenas más populares" },
  "Frequently seen with": { hi: "अक्सर इनके साथ", fr: "Souvent avec", ru: "Часто снимается с", es: "Suele aparecer con" },
  "All scenes with {name}": { hi: "{name} के सभी दृश्य", fr: "Toutes les scènes avec {name}", ru: "Все сцены с {name}", es: "Todas las escenas con {name}" },
  "Add to favourites": { hi: "पसंदीदा में जोड़ें", fr: "Ajouter aux favoris", ru: "В избранное", es: "Añadir a favoritas" },
  Favourite: { hi: "पसंदीदा", fr: "Favori", ru: "В избранном", es: "Favorita" },

  // my list
  "Favourite pornstars": { hi: "पसंदीदा पोर्नस्टार", fr: "Pornstars favorites", ru: "Избранные порнозвёзды", es: "Pornstars favoritas" },
  "Tap “Add to favourites” on a model's page to follow her.": {
    hi: "किसी मॉडल को फ़ॉलो करने के लिए उसके पेज पर “पसंदीदा में जोड़ें” दबाएँ।",
    fr: "Appuyez sur « Ajouter aux favoris » sur la page d'un modèle pour la suivre.",
    ru: "Нажмите «В избранное» на странице модели, чтобы следить за ней.",
    es: "Pulsa «Añadir a favoritas» en la página de una modelo para seguirla.",
  },
  "Saved for later": { hi: "बाद के लिए सहेजे गए", fr: "À regarder plus tard", ru: "Отложенные", es: "Guardado para después" },
  "Nothing saved yet. Tap “+ My list” on any scene.": {
    hi: "अभी कुछ सहेजा नहीं गया। किसी भी दृश्य पर “+ मेरी सूची” दबाएँ।",
    fr: "Rien pour l'instant. Appuyez sur « + Ma liste » sur une scène.",
    ru: "Пока ничего нет. Нажмите «+ Мой список» на любой сцене.",
    es: "Aún no hay nada. Pulsa «+ Mi lista» en cualquier escena.",
  },
  "Liked scenes": { hi: "पसंद किए दृश्य", fr: "Scènes aimées", ru: "Понравившиеся сцены", es: "Escenas que te gustan" },
  "Tap the heart on a scene to like it.": {
    hi: "किसी दृश्य को पसंद करने के लिए दिल दबाएँ।",
    fr: "Appuyez sur le cœur d'une scène pour l'aimer.",
    ru: "Нажмите на сердечко, чтобы отметить сцену.",
    es: "Pulsa el corazón de una escena para marcarla.",
  },

  // reels
  "▶ Reel": { hi: "▶ रील", fr: "▶ Reel", ru: "▶ Рилс", es: "▶ Reel" },
  "Watch reels": { hi: "रील्स देखें", fr: "Voir les reels", ru: "Смотреть рилсы", es: "Ver reels" },
  "▶ Watch full scene": { hi: "▶ पूरा दृश्य देखें", fr: "▶ Voir la scène entière", ru: "▶ Смотреть всю сцену", es: "▶ Ver escena completa" },
  Play: { hi: "चलाएँ", fr: "Lire", ru: "Воспроизвести", es: "Reproducir" },
  Mute: { hi: "आवाज़ बंद", fr: "Couper le son", ru: "Без звука", es: "Silenciar" },
  Unmute: { hi: "आवाज़ चालू", fr: "Activer le son", ru: "Включить звук", es: "Activar sonido" },
  "Reels are not available for your account.": {
    hi: "आपके खाते के लिए रील्स उपलब्ध नहीं हैं।",
    fr: "Les reels ne sont pas disponibles pour votre compte.",
    ru: "Рилсы недоступны для вашего аккаунта.",
    es: "Los reels no están disponibles para tu cuenta.",
  },
  "No reels yet.": { hi: "अभी कोई रील नहीं है।", fr: "Aucun reel pour l'instant.", ru: "Рилсов пока нет.", es: "Aún no hay reels." },

  // search
  "Searching…": { hi: "खोज रहे हैं…", fr: "Recherche…", ru: "Поиск…", es: "Buscando…" },
  "Nothing found for “{q}”.": {
    hi: "“{q}” के लिए कुछ नहीं मिला।",
    fr: "Aucun résultat pour « {q} ».",
    ru: "По запросу «{q}» ничего не найдено.",
    es: "No hay resultados para «{q}».",
  },
  // comments
  Comments: { hi: "टिप्पणियाँ", fr: "Commentaires", ru: "Комментарии", es: "Comentarios" },
  "Add a comment…": { hi: "टिप्पणी लिखें…", fr: "Ajouter un commentaire…", ru: "Написать комментарий…", es: "Añade un comentario…" },
  Post: { hi: "भेजें", fr: "Publier", ru: "Отправить", es: "Publicar" },
  Delete: { hi: "हटाएँ", fr: "Supprimer", ru: "Удалить", es: "Eliminar" },
  "on a reel": { hi: "रील पर", fr: "sur un reel", ru: "к рилсу", es: "en un reel" },
  "Be the first to comment.": {
    hi: "सबसे पहले टिप्पणी करें।",
    fr: "Soyez le premier à commenter.",
    ru: "Оставьте первый комментарий.",
    es: "Sé el primero en comentar.",
  },
  "Show more comments": { hi: "और टिप्पणियाँ", fr: "Plus de commentaires", ru: "Ещё комментарии", es: "Más comentarios" },
  Close: { hi: "बंद करें", fr: "Fermer", ru: "Закрыть", es: "Cerrar" },
  "just now": { hi: "अभी", fr: "à l'instant", ru: "только что", es: "ahora mismo" },
  "See all results": { hi: "सभी परिणाम देखें", fr: "Voir tous les résultats", ru: "Все результаты", es: "Ver todos los resultados" },
};

/** The current language's version of an English phrase, with {name} filled in. */
export function t(s: string, vars?: Record<string, string | number>): string {
  const tr = current === "en" ? s : dict[s]?.[current] ?? s;
  if (!vars) return tr;
  return tr.replace(/\{(\w+)\}/g, (m, k) => (k in vars ? String(vars[k]) : m));
}

type Forms = { one: string; few?: string; many?: string; other: string };

const plurals: Record<string, Record<Lang, Forms>> = {
  scene: {
    en: { one: "{n} scene", other: "{n} scenes" },
    hi: { one: "{n} दृश्य", other: "{n} दृश्य" },
    fr: { one: "{n} scène", other: "{n} scènes" },
    ru: { one: "{n} сцена", few: "{n} сцены", many: "{n} сцен", other: "{n} сцены" },
    es: { one: "{n} escena", other: "{n} escenas" },
  },
  model: {
    en: { one: "{n} model", other: "{n} models" },
    hi: { one: "{n} मॉडल", other: "{n} मॉडल" },
    fr: { one: "{n} modèle", other: "{n} modèles" },
    ru: { one: "{n} модель", few: "{n} модели", many: "{n} моделей", other: "{n} модели" },
    es: { one: "{n} modelo", other: "{n} modelos" },
  },
  rating: {
    en: { one: "{n} rating", other: "{n} ratings" },
    hi: { one: "{n} रेटिंग", other: "{n} रेटिंग" },
    fr: { one: "{n} note", other: "{n} notes" },
    ru: { one: "{n} оценка", few: "{n} оценки", many: "{n} оценок", other: "{n} оценки" },
    es: { one: "{n} valoración", other: "{n} valoraciones" },
  },
  view: {
    en: { one: "{n} view", other: "{n} views" },
    hi: { one: "{n} बार देखा गया", other: "{n} बार देखा गया" },
    fr: { one: "{n} vue", other: "{n} vues" },
    ru: { one: "{n} просмотр", few: "{n} просмотра", many: "{n} просмотров", other: "{n} просмотра" },
    es: { one: "{n} visualización", other: "{n} visualizaciones" },
  },
  comment: {
    en: { one: "{n} comment", other: "{n} comments" },
    hi: { one: "{n} टिप्पणी", other: "{n} टिप्पणियाँ" },
    fr: { one: "{n} commentaire", other: "{n} commentaires" },
    ru: { one: "{n} комментарий", few: "{n} комментария", many: "{n} комментариев", other: "{n} комментария" },
    es: { one: "{n} comentario", other: "{n} comentarios" },
  },
  together: {
    en: { one: "{n} scene together", other: "{n} scenes together" },
    hi: { one: "{n} दृश्य साथ में", other: "{n} दृश्य साथ में" },
    fr: { one: "{n} scène ensemble", other: "{n} scènes ensemble" },
    ru: { one: "{n} общая сцена", few: "{n} общие сцены", many: "{n} общих сцен", other: "{n} общие сцены" },
    es: { one: "{n} escena juntas", other: "{n} escenas juntas" },
  },
};

/** "3 scenes", "3 сцены", "5 сцен"… with the right form for the number. */
export function count(n: number, what: "scene" | "model" | "rating" | "together" | "view" | "comment"): string {
  const forms = plurals[what][current];
  const rule = new Intl.PluralRules(locale()).select(n) as keyof Forms;
  const form = forms[rule] ?? forms.other;
  return form.replace("{n}", n.toLocaleString(locale()));
}

/** A short number: 950, 1.2K, 3.4M (in the chosen language's style). */
export function short(n: number): string {
  try {
    return new Intl.NumberFormat(locale(), { notation: "compact", maximumFractionDigits: 1 }).format(n);
  } catch {
    return String(n);
  }
}

/** "1.2K views", with the plural form chosen by the full number. */
export function views(n: number): string {
  return count(n, "view").replace(n.toLocaleString(locale()), short(n));
}

/** "5 minutes ago", "yesterday"… in the chosen language. */
export function ago(date: string | Date): string {
  const then = new Date(date).getTime();
  const secs = Math.round((then - Date.now()) / 1000);
  if (Math.abs(secs) < 45) return t("just now");
  const rtf = new Intl.RelativeTimeFormat(locale(), { numeric: "auto" });
  const steps: [Intl.RelativeTimeFormatUnit, number][] = [
    ["minute", 60],
    ["hour", 3600],
    ["day", 86400],
    ["week", 604800],
    ["month", 2629800],
    ["year", 31557600],
  ];
  let unit: Intl.RelativeTimeFormatUnit = "minute";
  let size = 60;
  for (const [u, s] of steps) {
    if (Math.abs(secs) >= s) {
      unit = u;
      size = s;
    }
  }
  return rtf.format(Math.round(secs / size), unit);
}

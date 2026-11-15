// Узбекский — основной язык (SPEC): ключи берутся отсюда, ru и en обязаны иметь ровно те же.
export const uz = {
  'app.loading': 'Yuklanmoqda…',
  'app.error': 'Nimadir xato ketdi. Qayta urinib ko‘ring.',
  'app.retry': 'Qayta urinish',
  'app.outside': 'Ilovani Telegram ichida oching.',
  'app.expired': 'Sessiya eskirdi. Ilovani yopib, qayta oching.',

  'tab.home': 'Asosiy',
  'tab.exams': 'Imtihon',
  'tab.rating': 'Reyting',
  'tab.community': 'Jamoa',
  'tab.profile': 'Profil',

  'onb.step': '{n}/{total}',
  'onb.goal.title': 'Maqsadingiz qaysi ball?',
  'onb.goal.hint': 'Keyin profilda o‘zgartirish mumkin.',
  'onb.date.title': 'Imtihon qachon?',
  'onb.date.hint': 'Tayyorgarlik rejasini shu sanaga moslaymiz.',
  'onb.date.unknown': 'Hali bilmayman',
  'onb.next': 'Davom etish',
  'onb.back': 'Orqaga',
  'onb.start': 'Boshlash',

  'home.greeting': 'Salom, {name}!',
  'home.goal': 'Maqsad',
  'home.daysLeft': 'Imtihongacha {days} kun',
  'home.examToday': 'Imtihon bugun. Omad!',
  'home.noDate': 'Imtihon sanasi belgilanmagan',
  'home.soon': 'Bu yerda Shanba Mock, bugungi mashqlar va lenta bo‘ladi.',

  'soon.title': 'Tez orada',
  'exams.soon': 'Writing va Speaking tekshiruvi, Reading, Listening va to‘liq mock.',
  'rating.soon': 'Mock natijalari bo‘yicha reyting va do‘stlar bilan duellar.',
  'community.soon': 'Guruhlar, o‘qituvchilar, uy vazifalari va challenjlar.',

  'profile.goal': 'Maqsad ball',
  'profile.examDate': 'Imtihon sanasi',
  'profile.notSet': 'Belgilanmagan',
  'profile.language': 'Til',
  'profile.save': 'Saqlash',
  'profile.saved': 'Saqlandi',
  'profile.disclaimer': 'IELTS-style mock. British Council, IDP va Cambridge bilan bog‘liq emasmiz.',

  'error.date': 'Sana o‘tib ketgan yoki juda uzoq.',
} as const

export type Key = keyof typeof uz

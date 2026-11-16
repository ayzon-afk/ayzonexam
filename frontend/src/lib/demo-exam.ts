// Демо-задания этапа 12: тексты, вопросы и ответы — на английском, как на экзамене. Отдельно от
// lib/demo.ts, чтобы грузиться только с экранами практики, а не при открытии приложения.
import type { Key } from './i18n/index.svelte'

export type Q = { q: string; options: string[]; answer: number }

const TFNG = ['True', 'False', 'Not Given']

export const reading = {
  title: 'The Paper Makers of Samarkand',
  text: `For several centuries Samarkand was famous across Asia and Europe for its paper. According to a popular story, the technique reached the city in the eighth century, when Chinese craftsmen shared their knowledge after a battle. Whatever its origin, Samarkand paper was soon praised for being smooth, strong and pleasant to write on.

The paper was made from the bark of the mulberry tree. Workers soaked the bark in water, beat it into a soft pulp and spread it thinly on frames to dry. Finally, the sheets were polished with a stone until they shone. The process required patience: a skilled master could produce only a few hundred sheets a day.

By the nineteenth century, cheaper factory paper had almost replaced the traditional craft. In the 1990s, however, a local master, Zarif Mukhtarov, decided to revive it. Using old descriptions and years of experiments, he rebuilt a water-powered mill in the village of Konigil. Today the workshop attracts thousands of tourists, and its paper is used by artists, restorers and bookbinders.`,
  questions: [
    { q: 'Samarkand paper was known outside Asia.', options: TFNG, answer: 0 },
    { q: 'The paper was made from the leaves of the mulberry tree.', options: TFNG, answer: 1 },
    { q: 'Polishing the sheets was the first stage of the process.', options: TFNG, answer: 1 },
    { q: 'Samarkand paper was more expensive than Chinese paper.', options: TFNG, answer: 2 },
    { q: 'The mill in Konigil uses water power.', options: TFNG, answer: 0 },
    { q: 'A skilled master could make…', options: ['thousands of sheets a day', 'a few hundred sheets a day', 'a few sheets a week'], answer: 1 },
    { q: 'Why did the traditional craft almost disappear?', options: ['A war destroyed the mills', 'Mulberry trees died out', 'Factory paper was cheaper'], answer: 2 },
    { q: 'How did Zarif Mukhtarov learn the old method?', options: ['From his grandfather', 'From old descriptions and experiments', 'At a factory in China'], answer: 1 },
    { q: 'What happened to the bark after it was soaked?', options: ['It was dried in the sun', 'It was beaten into pulp', 'It was polished with a stone'], answer: 1 },
    { q: 'Who uses Konigil paper today?', options: ['Only tourists', 'Paper factories', 'Artists, restorers and bookbinders'], answer: 2 },
  ] as Q[],
}

export const listening: Q[] = [
  { q: 'What does the caller want to book?', options: ['A city tour', 'A hotel room', 'A train ticket'], answer: 0 },
  { q: 'Which day does the tour leave?', options: ['Friday', 'Saturday', 'Sunday'], answer: 1 },
  { q: 'What time should the group meet?', options: ['8:30', '9:00', '9:30'], answer: 0 },
  { q: 'Where is the meeting point?', options: ['The museum', 'The main gate of the park', 'The bus station'], answer: 1 },
  { q: 'How much is a student ticket?', options: ['$12', '$15', '$20'], answer: 0 },
  { q: 'What should visitors bring?', options: ['A camera', 'Comfortable shoes', 'Lunch'], answer: 1 },
]

export const level: Q[] = [
  { q: 'She ___ in Tashkent since 2020.', options: ['lives', 'has lived', 'is living', 'lived'], answer: 1 },
  { q: 'If I ___ more time, I would travel more.', options: ['have', 'had', 'will have', 'am having'], answer: 1 },
  { q: 'The report was written ___ the manager.', options: ['from', 'with', 'by', 'of'], answer: 2 },
  { q: 'Closest meaning: “significant”', options: ['important', 'small', 'quick', 'strange'], answer: 0 },
  { q: 'He is used to ___ early.', options: ['get up', 'getting up', 'got up', 'gets up'], answer: 1 },
  { q: 'Neither the teacher nor the students ___ ready.', options: ['was', 'were', 'is', 'be'], answer: 1 },
  { q: 'Closest meaning: “decline”', options: ['decrease', 'choose', 'explain', 'arrive'], answer: 0 },
  { q: 'By the time we arrived, the film ___.', options: ['started', 'has started', 'had started', 'starts'], answer: 2 },
]

export const duelWords: Q[] = [
  { q: 'rapid', options: ['slow', 'quick', 'heavy', 'quiet'], answer: 1 },
  { q: 'enormous', options: ['huge', 'tiny', 'narrow', 'early'], answer: 0 },
  { q: 'reluctant', options: ['eager', 'unwilling', 'careful', 'famous'], answer: 1 },
  { q: 'obtain', options: ['lose', 'refuse', 'get', 'hide'], answer: 2 },
  { q: 'vital', options: ['essential', 'optional', 'boring', 'weak'], answer: 0 },
  { q: 'decline', options: ['grow', 'decrease', 'repeat', 'borrow'], answer: 1 },
  { q: 'precise', options: ['rough', 'exact', 'late', 'cheap'], answer: 1 },
  { q: 'abundant', options: ['rare', 'plentiful', 'empty', 'small'], answer: 1 },
]

// Примерный балл по доле верных ответов в коротком сете: 0 → 4.0, все → 9.0, шаг 0.5.
export function setBand(correct: number, total: number): number {
  return 4 + Math.round((correct / total) * 10) * 0.5
}

export const speaking = {
  parts: [
    'Do you work or are you a student? What do you like about it?',
    'Describe a place in your city that you like to visit. You should say: where it is, how often you go there, what you do there, and explain why you like it.',
    'Why do people enjoy visiting historical places? Has this changed in recent years?',
  ],
  band: 6.5,
  crit: [
    { name: 'Fluency and Coherence', band: 6.5 },
    { name: 'Lexical Resource', band: 6.0 },
    { name: 'Grammatical Range and Accuracy', band: 6.5 },
    { name: 'Pronunciation', band: 7.0 },
  ],
  transcript:
    'Well, I am a student at the moment, I study economics at university. What I like most is, um, the people, because my groupmates are very friendly and we often prepare for exams together…',
  notes: ['demo.sp1', 'demo.sp2'] as Key[],
}

export const peerEssay =
  'Nowadays many people believe that technology makes our life easier. In my opinion, this is true, but it also has some disadvantages. Firstly, phones and computers help us to communicate with friends who live far away. Secondly, we can find any information in internet very fast…'

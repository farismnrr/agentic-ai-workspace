/** Splits a typed command into words; text in single or double quotes stays one word. */
export function splitCommand(text: string): string[] {
  const words: string[] = [];
  let current = '';
  let quote = '';
  let started = false;

  for (const char of text.trim()) {
    if (quote !== '') {
      if (char === quote) quote = '';
      else current += char;
    } else if (char === '"' || char === "'") {
      quote = char;
      started = true;
    } else if (/\s/.test(char)) {
      if (started) words.push(current);
      current = '';
      started = false;
    } else {
      current += char;
      started = true;
    }
  }

  if (started) words.push(current);
  return words;
}

const quoteWord = (word: string): string => {
  if (word !== '' && !/[\s"']/.test(word)) return word;
  return word.includes('"') ? `'${word}'` : `"${word}"`;
};

/** The reverse of splitCommand, for showing a command in a text box. */
export const joinCommand = (words: string[]): string => words.map(quoteWord).join(' ');

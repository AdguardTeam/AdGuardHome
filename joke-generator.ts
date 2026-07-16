/**
 * Random Joke Generator
 * Fetches jokes from external APIs and provides utility functions
 */

interface JokeResponse {
  joke?: string;
  setup?: string;
  delivery?: string;
  error?: boolean;
  message?: string;
}

interface FormattedJoke {
  text: string;
  type: 'single' | 'two-part';
  source: string;
}

/**
 * Fetches a random joke from the JokeAPI
 * @returns Promise<FormattedJoke> - The formatted joke object
 */
export async function getRandomJoke(): Promise<FormattedJoke> {
  try {
    const response = await fetch('https://v2.jokeapi.dev/joke/Any');
    
    if (!response.ok) {
      throw new Error(`API responded with status: ${response.status}`);
    }

    const data: JokeResponse = await response.json();

    if (data.error) {
      throw new Error(data.message || 'Failed to fetch joke');
    }

    // Handle two-part joke (setup + delivery)
    if (data.setup && data.delivery) {
      return {
        text: `${data.setup}\n\n${data.delivery}`,
        type: 'two-part',
        source: 'JokeAPI'
      };
    }

    // Handle single-line joke
    if (data.joke) {
      return {
        text: data.joke,
        type: 'single',
        source: 'JokeAPI'
      };
    }

    throw new Error('Unexpected API response format');
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Error fetching joke:', errorMessage);
    throw error;
  }
}

/**
 * Fetches a random joke from a specific category
 * @param category - The joke category (e.g., 'programming', 'general', 'knock-knock')
 * @returns Promise<FormattedJoke> - The formatted joke object
 */
export async function getJokeByCategory(category: string): Promise<FormattedJoke> {
  try {
    const response = await fetch(`https://v2.jokeapi.dev/joke/${category}`);
    
    if (!response.ok) {
      throw new Error(`API responded with status: ${response.status}`);
    }

    const data: JokeResponse = await response.json();

    if (data.error) {
      throw new Error(`No jokes found for category: ${category}`);
    }

    if (data.setup && data.delivery) {
      return {
        text: `${data.setup}\n\n${data.delivery}`,
        type: 'two-part',
        source: 'JokeAPI'
      };
    }

    if (data.joke) {
      return {
        text: data.joke,
        type: 'single',
        source: 'JokeAPI'
      };
    }

    throw new Error('Unexpected API response format');
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Error fetching joke by category:', errorMessage);
    throw error;
  }
}

/**
 * Fetches multiple random jokes
 * @param count - Number of jokes to fetch (max 10)
 * @returns Promise<FormattedJoke[]> - Array of formatted jokes
 */
export async function getMultipleJokes(count: number = 3): Promise<FormattedJoke[]> {
  const maxCount = Math.min(Math.max(count, 1), 10); // Clamp between 1 and 10
  const jokes: FormattedJoke[] = [];

  try {
    for (let i = 0; i < maxCount; i++) {
      const joke = await getRandomJoke();
      jokes.push(joke);
      // Add small delay to avoid rate limiting
      await new Promise(resolve => setTimeout(resolve, 100));
    }
    return jokes;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : 'Unknown error';
    console.error('Error fetching multiple jokes:', errorMessage);
    throw error;
  }
}

/**
 * Prints a formatted joke to console
 * @param joke - The joke object to print
 */
export function printJoke(joke: FormattedJoke): void {
  console.log('\n' + '='.repeat(50));
  console.log(`📝 ${joke.source} - ${joke.type === 'two-part' ? 'Two-Part' : 'Single'} Joke`);
  console.log('='.repeat(50));
  console.log(joke.text);
  console.log('='.repeat(50) + '\n');
}

/**
 * Main function - demonstrates joke generator usage
 */
export async function main(): Promise<void> {
  try {
    console.log('🎭 Random Joke Generator - Powered by JokeAPI\n');

    // Get a single random joke
    console.log('Fetching a random joke...');
    const randomJoke = await getRandomJoke();
    printJoke(randomJoke);

    // Get a programming joke
    console.log('Fetching a programming joke...');
    const progJoke = await getJokeByCategory('Programming');
    printJoke(progJoke);

    // Get multiple jokes
    console.log('Fetching 3 random jokes...');
    const multipleJokes = await getMultipleJokes(3);
    multipleJokes.forEach((joke, index) => {
      console.log(`\n--- Joke ${index + 1} ---`);
      printJoke(joke);
    });
  } catch (error) {
    console.error('Error in main:', error);
    process.exit(1);
  }
}

// Run if this is the main module
if (require.main === module) {
  main();
}

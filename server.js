import express from "express";
import { pipeline, env } from "@xenova/transformers";
import Tokenizer from "sentence-tokenizer";

env.allowLocalModels = false;
// env.userBrowserCache = false;

class Pipeline {
    static task = "zero-shot-classification";
    static model = "Xenova/roberta-large-mnli";
    static instance = null;

    static async getInstance(progressCallback = null) {
        if (this.instance === null) {
            this.instance = await pipeline(this.task, this.model, { progress_callback: progressCallback });
        }
        return this.instance;
    }
}

const app = express();
app.use(express.json());

const SCORE_THRESHOLD = 0.4;
const SCORE_INCREMENT = 0.05;

const containsInterrogativeWord = (sentence) => {
    // List of common interrogative words
    const interrogativeWords = [
        "what", "how", "why", "when", "where", "who",
        "which", "is", "are", "can", "could", "would", "should"
    ];

    // Check if any interrogative word is present in the beginning of a sentence
    return interrogativeWords.some(word => sentence.toLowerCase().startsWith(word));
}

const endsWithQuestionMark = (sentence) => {
    // Check if the last character is a question mark
    return sentence.endsWith('?');
}

const normalizeText = (str) => {
    // Remove hashtags and newlines, then trim whitespace
    return str.replace(/#\w+/g, "") // remove hashtags
        .replace(/https?:\/\/[^\s]+|www\.[^\s]+/g, '') // remove urls
        .replace(/\n/g, " ") // replace newlines with space
        .replace(/\s+/g, " ") // replace multiple spaces with a single space
        .replace(/['"]/g, "") // remove single and double quotes
        .replace(/[~^&*[\]{}|<>]/g, "") // remove special chars
        .replace(/[\u{1F600}-\u{1F64F}]/gu, "") // remove emojis
        .replace(/[!?.,;:]{2,}/g, match => match[0]) // replace multiple instances of punctuation marks with a single one
        .replace(/[^\w\s.,!?;:]/g, '') // remove any other unwanted punctuation characters but keep basic ones: . , ! ? ; :
        .replace(/nostr:[a-zA-Z0-9]+/g, '[REFERENCE]') // replace bech32 entities references
        .trim()
        .toLowerCase();
}

const classifySentences = async (sentences, candidate_labels, hypothesis_template) => {
    const classifier = await Pipeline.getInstance();

    const classificationPromises = sentences.map(sentence =>
        classifier(sentence, candidate_labels, { hypothesis_template})
    )
    return await Promise.all(classificationPromises);
}

app.post("/classify-text", async (req, res) => {
    const text = normalizeText(req.body.text);
    console.log({text});

    const tokenizer = new Tokenizer();
    tokenizer.setEntry(text);

    const sentences = tokenizer.getSentences();

    console.log({sentences});

    const candidate_labels = ["question", "statement", "command", "exclamation"];
    const hypothesis_template = "This text is a {}.";

    const responses = await classifySentences(sentences, candidate_labels, hypothesis_template);

    const questions = responses.filter((response, index) => {
        const { labels, scores } = response;
        console.log('Response', {labels, scores})
        return labels[0] === "question" && scores[0] >= SCORE_THRESHOLD;
    }).map((response, index) => sentences[index].trim());

    res.json({ result: questions.length > 0 });
});

const PORT = 3006;
app.listen(PORT, () => {
    console.log(`Server is running on http://localhost:${PORT}`);
});

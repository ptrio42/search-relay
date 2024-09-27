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

const question_labels = ["clear factual question", "yes/no question", "open-ended question", "vague question", "contextual inquiry"];

app.use(express.json());

const cleanString = (str) => {
    // Remove hashtags and newlines, then trim whitespace
    return str.replace(/#\w+/g, '') // Remove hashtags
        .replace(/\n/g, ' ') // Replace newlines with space
        .replace(/\s+/g, ' ') // Replace multiple spaces with a single space
        .trim();
}

app.post("/classify-text", async (req, res) => {
    const text = cleanString(req.body.text);
    console.log({text});

    const tokenizer = new Tokenizer();
    tokenizer.setEntry(text);

    const sentences = tokenizer.getSentences();

    console.log({sentences});

    const classifier = await Pipeline.getInstance();
    const candidate_labels = [...question_labels, "statement", "command", "exclamation"];
    const hypothesis_template = "This text is a {}.";

    const questions = [];

    for (const sentence of sentences) {
        const response = await classifier(sentence, candidate_labels, {
            hypothesis_template: hypothesis_template
        });

        const {labels, score} = response;

        if (question_labels.includes(labels[0]) && score >= 0.4) {
            console.log("A question!", {response});
            questions.push([sentence.trim()]);
        }
    }

    res.json({ result: questions.length > 0 });
});

const PORT = 3006;
app.listen(PORT, () => {
    console.log(`Server is running on http://localhost:${PORT}`);
});

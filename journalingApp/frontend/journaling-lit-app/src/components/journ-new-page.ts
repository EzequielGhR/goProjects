import { LitElement, html, css,  } from 'lit';
import { customElement, state } from 'lit/decorators.js';
import { JournalService } from '../service/journal-service';
import { PageWithContent } from '../abstract/JournalTypes';


@customElement("journ-new-page")
export class JournNewPage extends LitElement {
    static styles = css`
        :host {
            position: fixed;
            top: 0;
            left: 0;
            width: 100vw;
            height: 100vh;
            background: rgba(0, 0, 0, 0.7);
            display: flex;
            justify-content: center;
            align-items: center;
        }

        .modal {
            background: white;
            width: 80%;
            height: 90%;
            border-radius: 8px;
            padding: 20px;
            box-shadow: 0 4px 10px rgba(0, 0, 0, 0.3);
            display: flex;
            flex-direction: column;
            box-sizing: border-box; /* Prevent overflow */
        }

        h2 {
            margin: 0;
            text-align: center;
        }

        input, textarea {
            width: calc(100% - 24px); /* Prevent overflow */
            font-size: 1.5rem;
            padding: 12px;
            margin: 10px auto;
            border: 1px solid #ddd;
            border-radius: 5px;
            box-sizing: border-box;
        }

        textarea {
            flex: 1;
            resize: none;
        }

        .buttons {
            display: flex;
            justify-content: space-between;
            margin-top: 10px;
        }

        button {
            padding: 12px;
            font-size: 1.2rem;
            cursor: pointer;
            border: none;
            border-radius: 5px;
            width: 48%;
            max-width: 180px; /* Prevent stretching */
            margin: 5px;
        }

        .save {
            background: #28a745;
            color: white;
        }

        .cancel {
            background: #dc3545;
            color: white;
        }

        button:hover {
            opacity: 0.8;
        }
    `;

    protected service = JournalService.newLocal();

    @state()
    private pageTitle = "";

    @state()
    private content = "";

    private async savePage() {
        if (!this.pageTitle.trim()) {
            alert("Title is required");
        }

        const page: PageWithContent = {id: "", path: "", title: this.pageTitle, content: this.content}
        await this.service.createPage(page);

        this.dispatchEvent(new CustomEvent("jl-close", { bubbles: true, composed: true }));
    }

    private closeModal() {
        this.dispatchEvent(new CustomEvent("jl-close", { bubbles: true, composed: true }));
    }

    render() {
        return html`
            <div class="modal">
                <h2>Create New Page</h2>
                <input type="text" placeholder="Title" .value=${this.pageTitle} @input=${(e: any) => this.pageTitle = e.target.value} />
                <textarea placeholder="Content" .value=${this.content} @input=${(e: any) => this.content = e.target.value}></textarea>
                <button class="save" @click=${this.savePage}>Save</button>
                <button class="cancel" @click=${this.closeModal}>Cancel</button>
            </div>
        `
    }
}
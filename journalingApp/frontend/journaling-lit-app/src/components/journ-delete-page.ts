import { LitElement, html, css,  } from 'lit';
import { customElement, property} from 'lit/decorators.js';
import { JournalService } from '../service/journal-service';
import { PageWithContent } from '../abstract/JournalTypes';


@customElement("journ-delete-page")
export class JournDeletePage extends LitElement {
    @property({ type: Object })
    private page?: PageWithContent;

    static styles = css`
        .modal {
            background: gray;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }

        .buttons {
            margin-top: 10px;
            display: flex;
            justify-content: space-around;
        }

        button {
            padding: 10px;
            font-size: 1rem;
            cursor: pointer;
            border: none;
            border-radius: 5px;
        }

        .delete {
            background: #dc3545;
            color: white;
        }

        .cancel {
            background: #6c757d;
            color: white;
        }
    `;

    protected service = JournalService.newLocal();

    private confirmDelete() {
        if (!this.page) {
            return;
        }

        this.service.deletePage(this.page.id)
        this.dispatchEvent(new CustomEvent("jl-close", { bubbles: true, composed: true }));
    }

    private closeDialog() {
        this.dispatchEvent(new Event('jl-close', { bubbles: true, composed: true }));
    }

    render() {
        return html`
            <div class="modal">
                <h3>Delete Page?</h3>
                <p>Are you sure you want to delete <b>${this.page?.title}</b>?</p>
                <div class="buttons">
                    <button class="delete" @click="${this.confirmDelete}">Delete</button>
                    <button class="cancel" @click="${this.closeDialog}">Cancel</button>
                </div>
            </div>
        `
    }
}
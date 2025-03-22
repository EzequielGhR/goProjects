import { LitElement, html } from "lit";
import { customElement } from "lit/decorators.js";
import { JournalService } from "./service/journal-service";
import "./components/journ-page-list"

@customElement("app-root")
export class AppRoot extends LitElement {
    protected service = JournalService.newLocal();

    render() {
        return html`
            <journ-page-list></journ-page-list>
        `;
    }
}

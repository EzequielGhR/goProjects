import { provide } from "@lit/context";
import { LitElement, html } from "lit";
import { customElement } from "lit/decorators.js";
import { appContext, AppContext } from "./data/context";
import { JournalService } from "./service/journal-service";
import "./components/journ-page-list"

@customElement("app-root")
export class AppRoot extends LitElement {
    @provide({ context: appContext })   
    context: AppContext = { journalService: new JournalService("http://localhost:8080") };

    render() {
        console.log(" Main render:", this.context)
        return html`
            <journ-page-list></journ-page-list>
        `;
    }
}

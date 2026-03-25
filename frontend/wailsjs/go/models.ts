export namespace events {
	
	export class EventCategoryEntry {
	    type: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new EventCategoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.label = source["label"];
	    }
	}
	export class EventCategory {
	    name: string;
	    events: EventCategoryEntry[];
	
	    static createFrom(source: any = {}) {
	        return new EventCategory(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.events = this.convertValues(source["events"], EventCategoryEntry);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class AppConfig {
	    logPath: string;
	    apiEndpoint: string;
	    environment: string;
	    connected: boolean;
	    handle: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.logPath = source["logPath"];
	        this.apiEndpoint = source["apiEndpoint"];
	        this.environment = source["environment"];
	        this.connected = source["connected"];
	        this.handle = source["handle"];
	    }
	}
	export class ConnectionStatus {
	    connected: boolean;
	    handle: string;
	    endpoint: string;
	    connectedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.handle = source["handle"];
	        this.endpoint = source["endpoint"];
	        this.connectedAt = source["connectedAt"];
	    }
	}
	export class EventEntry {
	    type: string;
	    source: string;
	    timestamp: string;
	    data: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new EventEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.source = source["source"];
	        this.timestamp = source["timestamp"];
	        this.data = source["data"];
	    }
	}
	export class FriendEntry {
	    account_id: string;
	    nickname: string;
	    display_name: string;
	    presence: string;
	    activity_state: string;
	    activity_detail: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new FriendEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.account_id = source["account_id"];
	        this.nickname = source["nickname"];
	        this.display_name = source["display_name"];
	        this.presence = source["presence"];
	        this.activity_state = source["activity_state"];
	        this.activity_detail = source["activity_detail"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class StatusInfo {
	    playerHandle: string;
	    currentShip: string;
	    location: string;
	    jurisdiction: string;
	    tailerActive: boolean;
	    eventCount: number;
	    lastEvent: string;
	    connected: boolean;
	    handle: string;
	    environment: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.playerHandle = source["playerHandle"];
	        this.currentShip = source["currentShip"];
	        this.location = source["location"];
	        this.jurisdiction = source["jurisdiction"];
	        this.tailerActive = source["tailerActive"];
	        this.eventCount = source["eventCount"];
	        this.lastEvent = source["lastEvent"];
	        this.connected = source["connected"];
	        this.handle = source["handle"];
	        this.environment = source["environment"];
	    }
	}

}

export namespace updater {
	
	export class ReleaseInfo {
	    version: string;
	    name: string;
	    url: string;
	    downloadUrl: string;
	    installerUrl: string;
	    publishedAt: string;
	    hasUpdate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ReleaseInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.name = source["name"];
	        this.url = source["url"];
	        this.downloadUrl = source["downloadUrl"];
	        this.installerUrl = source["installerUrl"];
	        this.publishedAt = source["publishedAt"];
	        this.hasUpdate = source["hasUpdate"];
	    }
	}

}


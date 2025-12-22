export namespace clip_helper {
	
	export class ClipboardItem {
	    type: string;
	    text?: string;
	    image?: number[];
	    files?: string[];
	    isSingleFile?: boolean;
	    singleFileName?: string;
	    singleFileMime?: string;
	    singleFileSize?: number;
	    singleFileThumb?: string;
	
	    static createFrom(source: any = {}) {
	        return new ClipboardItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.text = source["text"];
	        this.image = source["image"];
	        this.files = source["files"];
	        this.isSingleFile = source["isSingleFile"];
	        this.singleFileName = source["singleFileName"];
	        this.singleFileMime = source["singleFileMime"];
	        this.singleFileSize = source["singleFileSize"];
	        this.singleFileThumb = source["singleFileThumb"];
	    }
	}

}

export namespace main {
	
	export class ChatMessage {
	    id: string;
	    roomId: string;
	    userId: string;
	    userName: string;
	    message: string;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.roomId = source["roomId"];
	        this.userId = source["userId"];
	        this.userName = source["userName"];
	        this.message = source["message"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class DroppedFilePayload {
	    name: string;
	    rel: string;
	    data: string;
	
	    static createFrom(source: any = {}) {
	        return new DroppedFilePayload(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.rel = source["rel"];
	        this.data = source["data"];
	    }
	}
	export class Item {
	    id: string;
	    type: string;
	    data: any;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.data = source["data"];
	    }
	}
	export class Operation {
	    id: string;
	    parentId: string;
	    parentHash?: string;
	    hash: string;
	    opType: string;
	    itemId: string;
	    item?: Item;
	    timestamp: number;
	    userId?: string;
	    userName?: string;
	
	    static createFrom(source: any = {}) {
	        return new Operation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentId = source["parentId"];
	        this.parentHash = source["parentHash"];
	        this.hash = source["hash"];
	        this.opType = source["opType"];
	        this.itemId = source["itemId"];
	        this.item = this.convertValues(source["item"], Item);
	        this.timestamp = source["timestamp"];
	        this.userId = source["userId"];
	        this.userName = source["userName"];
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
	export class Room {
	    id: string;
	    name: string;
	    ownerId: string;
	    userIds: string[];
	    approvedUserIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new Room(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.ownerId = source["ownerId"];
	        this.userIds = source["userIds"];
	        this.approvedUserIds = source["approvedUserIds"];
	    }
	}
	export class User {
	    id: string;
	    name: string;
	    roomId?: string;
	    isOnline: boolean;
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.roomId = source["roomId"];
	        this.isOnline = source["isOnline"];
	    }
	}

}


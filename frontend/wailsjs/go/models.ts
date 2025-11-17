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
	export class ClipboardItem {
	    type: string;
	    text?: string;
	    image?: number[];
	
	    static createFrom(source: any = {}) {
	        return new ClipboardItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.text = source["text"];
	        this.image = source["image"];
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
	    opType: string;
	    itemId: string;
	    item?: Item;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new Operation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentId = source["parentId"];
	        this.opType = source["opType"];
	        this.itemId = source["itemId"];
	        this.item = this.convertValues(source["item"], Item);
	        this.timestamp = source["timestamp"];
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
	    userIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new Room(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.userIds = source["userIds"];
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


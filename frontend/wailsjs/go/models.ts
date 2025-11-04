export namespace main {
	
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


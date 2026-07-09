/* tslint:disable */
/* eslint-disable */
// Generated using typescript-generator version 2.22.595 on 2026-05-21 15:36:52.

export namespace Digit {

    interface BusinessServiceRequest {
        RequestInfo: RequestInfo;
        BusinessServices: BusinessService[];
    }

    interface BusinessServiceResponse {
        ResponseInfo: ResponseInfo;
        BusinessServices: BusinessService[];
    }

    /**
     * Contract class to receive request. Array of TradeLicense items are used in case of create, whereas single TradeLicense item is used for update
     */
    interface ProcessInstanceRequest {
        RequestInfo: RequestInfo;
        ProcessInstances: ProcessInstance[];
    }

    /**
     * Contract class to send response. Array of TradeLicense items are used in case of search results or response for create, whereas single TradeLicense item is used for update
     */
    interface ProcessInstanceResponse {
        ResponseInfo: ResponseInfo;
        ProcessInstances: ProcessInstance[];
        totalCount: number;
    }

    interface UserSearchRequest {
        RequestInfo: RequestInfo;
        uuid: string[];
        id: string[];
        userName: string;
        name: string;
        mobileNumber: string;
        aadhaarNumber: string;
        pan: string;
        emailId: string;
        fuzzyLogic: boolean;
        active: boolean;
        tenantId: string;
        pageSize: number;
        pageNumber: number;
        sort: string[];
        userType: string;
        roleCodes: string[];
    }

    interface UserDetailResponse {
        responseInfo: ResponseInfo;
        user: CommonUser[];
    }

    /**
     * A Object holds the
     */
    interface BusinessService {
        tenantId: string;
        uuid: string;
        businessService: string;
        business: string;
        getUri: string;
        postUri: string;
        businessServiceSla: number;
        states: State[];
        auditDetails: AuditDetails;
    }

    /**
     * A Object holds the basic data for a Trade License
     */
    interface ProcessInstance {
        id: string;
        tenantId: string;
        businessService: string;
        businessId: string;
        action: string;
        moduleName: string;
        state: State;
        comment: string;
        documents: Document[];
        assigner: CommonUser;
        assignes: CommonUser[];
        nextActions: Action[];
        stateSla: number;
        businesssServiceSla: number;
        previousStatus: string;
        entity: any;
        auditDetails: AuditDetails;
        rating: number;
        escalated: boolean;
    }

    /**
     * A Object holds the basic data for a Trade License
     */
    interface State {
        auditDetails: AuditDetails;
        uuid: string;
        tenantId: string;
        businessServiceId: string;
        sla: number;
        state: string;
        applicationStatus: string;
        docUploadRequired: boolean;
        isStartState: boolean;
        isTerminateState: boolean;
        isStateUpdatable: boolean;
        actions: Action[];
    }

    /**
     * Collection of audit related fields used by most models
     */
    interface AuditDetails {
        createdBy: string;
        lastModifiedBy: string;
        createdTime: number;
        lastModifiedTime: number;
    }

    /**
     * A Object holds the basic data for a Trade License
     */
    interface Document {
        id: string;
        tenantId: string;
        documentType: string;
        fileStoreId: string;
        documentUid: string;
        auditDetails: AuditDetails;
    }

    /**
     * A Object holds the basic data for a Trade License
     */
    interface Action {
        auditDetails: AuditDetails;
        uuid: string;
        tenantId: string;
        currentState: string;
        action: string;
        nextState: string;
        roles: string[];
        active: boolean;
    }

}

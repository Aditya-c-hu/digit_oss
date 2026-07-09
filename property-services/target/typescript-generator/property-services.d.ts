/* tslint:disable */
/* eslint-disable */
// Generated using typescript-generator version 2.22.595 on 2026-05-28 17:42:50.

export namespace Digit {

    interface AssessmentRequest {
        RequestInfo: RequestInfo;
        Assessment: Assessment;
    }

    interface AssessmentResponse {
        ResponseInfo: ResponseInfo;
        Assessments: Assessment[];
    }

    interface PropertyRequest {
        RequestInfo: RequestInfo;
        Property: Property;
    }

    interface PropertyResponse {
        ResponseInfo: ResponseInfo;
        Properties: Property[];
        count: number;
    }

    interface Assessment {
        id: string;
        tenantId: string;
        assessmentNumber: string;
        financialYear: string;
        propertyId: string;
        assessmentDate: number;
        status: Status;
        source: Source;
        unitUsageList: UnitUsage[];
        documents: Document[];
        additionalDetails: any;
        channel: Channel;
        auditDetails: AuditDetails;
        workflow: ProcessInstance;
    }

    interface Property extends PropertyInfo {
        acknowldgementNumber: string;
        propertyType: string;
        ownershipCategory: string;
        owners: OwnerInfo[];
        institution: Institution;
        creationReason: CreationReason;
        usageCategory: string;
        noOfFloors: number;
        landArea: number;
        superBuiltUpArea: number;
        source: PropertySource;
        channel: Channel;
        documents: Document[];
        units: Unit[];
        additionalDetails: any;
        auditDetails: AuditDetails;
        workflow: ProcessInstance;
        AlternateUpdated: boolean;
        isOldDataEncryptionRequest: boolean;
    }

    interface UnitUsage {
        id: string;
        tenantId: string;
        unitId: string;
        usageCategory: string;
        occupancyType: string;
        occupancyDate: number;
        auditDetails: AuditDetails;
    }

    interface Document {
        id: string;
        documentType: string;
        fileStoreId: string;
        documentUid: string;
        auditDetails: AuditDetails;
        status: Status;
    }

    interface AuditDetails {
        createdBy: string;
        lastModifiedBy: string;
        createdTime: number;
        lastModifiedTime: number;
    }

    /**
     * A Object holds the basic data for a Trade License
     */
    interface ProcessInstance {
        notificationAction: string;
        id: string;
        tenantId: string;
        businessService: string;
        businessId: string;
        action: string;
        moduleName: string;
        state: State;
        comment: string;
        documents: Document[];
        assignes: OwnerInfo[];
    }

    interface Address {
        tenantId: string;
        doorNo: string;
        plotNo: string;
        id: string;
        landmark: string;
        city: string;
        district: string;
        region: string;
        state: string;
        country: string;
        pincode: string;
        buildingName: string;
        street: string;
        locality: Locality;
        geoLocation: GeoLocation;
        additionalDetails: any;
    }

    interface OwnerInfo extends User {
        ownerInfoUuid: string;
        isPrimaryOwner: boolean;
        ownerShipPercentage: number;
        ownerType: string;
        institutionId: string;
        status: Status;
        documents: Document[];
        relationship: Relationship;
    }

    interface Institution {
        id: string;
        tenantId: string;
        name: string;
        type: string;
        designation: string;
        nameOfAuthorizedPerson: string;
        additionalDetails: any;
    }

    interface Unit {
        id: string;
        tenantId: string;
        floorNo: number;
        unitType: string;
        usageCategory: string;
        occupancyType: string;
        active: boolean;
        occupancyDate: number;
        constructionDetail: ConstructionDetail;
        additionalDetails: any;
        auditDetails: AuditDetails;
        arv: number;
    }

    interface PropertyInfo {
        id: string;
        propertyId: string;
        surveyId: string;
        linkedProperties: string[];
        tenantId: string;
        accountId: string;
        oldPropertyId: string;
        status: Status;
        address: Address;
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

    interface Locality {
        code: string;
        name: string;
        label: string;
        latitude: string;
        longitude: string;
        area: string;
        children: Locality[];
        materializedPath: string;
    }

    interface GeoLocation {
        latitude: number;
        longitude: number;
    }

    interface Role {
        id: number;
        name: string;
        code: string;
        tenantId: string;
    }

    interface User {
        id: number;
        uuid: string;
        userName: string;
        password: string;
        salutation: string;
        name: string;
        gender: string;
        mobileNumber: string;
        emailId: string;
        altContactNumber: string;
        pan: string;
        aadhaarNumber: string;
        permanentAddress: string;
        permanentCity: string;
        permanentPinCode: string;
        correspondenceCity: string;
        correspondencePinCode: string;
        correspondenceAddress: string;
        active: boolean;
        dob: number;
        pwdExpiryDate: number;
        locale: string;
        type: string;
        signature: string;
        accountLocked: boolean;
        roles: Role[];
        fatherOrHusbandName: string;
        bloodGroup: string;
        identificationMark: string;
        photo: string;
        createdBy: string;
        createdDate: number;
        lastModifiedBy: string;
        lastModifiedDate: number;
        tenantId: string;
        alternatemobilenumber: string;
    }

    interface ConstructionDetail {
        carpetArea: number;
        builtUpArea: number;
        plinthArea: number;
        superBuiltUpArea: number;
        constructionType: string;
        constructionDate: number;
        dimensions: any;
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
    }

    type Status = "ACTIVE" | "INACTIVE" | "INWORKFLOW" | "CANCELLED" | "REJECTED";

    type Source = "MUNICIPAL_RECORDS" | "WEBAPP" | "MOBILEAPP" | "FIELD_SURVEY";

    type Channel = "SYSTEM" | "CFC_COUNTER" | "CITIZEN" | "DATA_ENTRY" | "MIGRATION";

    type CreationReason = "CREATE" | "UPDATE" | "MUTATION" | "LEGACY_ENTRY" | "BIFURCATION" | "AMALGAMATION" | "SUBDIVISION" | "DATA_UPLOAD";

    type Relationship = "FATHER" | "HUSBAND";

}

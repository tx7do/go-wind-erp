// DO NOT EDIT. This is code generated via package:intl/generate_localized.dart
// This is a library that provides messages for a en_US locale. All the
// messages from the main program should be duplicated here with the same
// function name.

// Ignore issues from commonly used lints in this file.
// ignore_for_file:unnecessary_brace_in_string_interps, unnecessary_new
// ignore_for_file:prefer_single_quotes,comment_references, directives_ordering
// ignore_for_file:annotate_overrides,prefer_generic_function_type_aliases
// ignore_for_file:unused_import, file_names, avoid_escaping_inner_quotes
// ignore_for_file:unnecessary_string_interpolations, unnecessary_string_escapes

import 'package:intl/intl.dart';
import 'package:intl/message_lookup_by_library.dart';

final messages = new MessageLookup();

typedef String MessageIfAbsent(String messageStr, List<dynamic> args);

class MessageLookup extends MessageLookupByLibrary {
  String get localeName => 'en_US';

  final messages = _notInlinedMessages(_notInlinedMessages);
  static Map<String, Function> _notInlinedMessages(_) => <String, Function>{
    "appName": MessageLookupByLibrary.simpleMessage("GoWind ERP"),
    "appearance": MessageLookupByLibrary.simpleMessage("Appearance"),
    "approvalActionRejected": MessageLookupByLibrary.simpleMessage(
      "Rejected: state changed or no permission",
    ),
    "approvalApprove": MessageLookupByLibrary.simpleMessage("Approve"),
    "approvalCancel": MessageLookupByLibrary.simpleMessage("Cancel Request"),
    "approvalCancelConfirm": MessageLookupByLibrary.simpleMessage(
      "Cancel this approval request?",
    ),
    "approvalComment": MessageLookupByLibrary.simpleMessage("Comment"),
    "approvalEmpty": MessageLookupByLibrary.simpleMessage(
      "No approval requests",
    ),
    "approvalFilterAll": MessageLookupByLibrary.simpleMessage("All"),
    "approvalReject": MessageLookupByLibrary.simpleMessage("Reject"),
    "approvalStatusApproved": MessageLookupByLibrary.simpleMessage("Approved"),
    "approvalStatusCancelled": MessageLookupByLibrary.simpleMessage(
      "Cancelled",
    ),
    "approvalStatusPending": MessageLookupByLibrary.simpleMessage("Pending"),
    "approvalStatusRejected": MessageLookupByLibrary.simpleMessage("Rejected"),
    "backToHome": MessageLookupByLibrary.simpleMessage("Back to Home"),
    "cancel": MessageLookupByLibrary.simpleMessage("Cancel"),
    "comingSoon": MessageLookupByLibrary.simpleMessage(
      "Module under development",
    ),
    "confirm": MessageLookupByLibrary.simpleMessage("Confirm"),
    "dark": MessageLookupByLibrary.simpleMessage("Dark"),
    "darkMode": MessageLookupByLibrary.simpleMessage("Dark Mode"),
    "errorOccurred": MessageLookupByLibrary.simpleMessage("Error Occurred!"),
    "followSystem": MessageLookupByLibrary.simpleMessage("System"),
    "inbound": MessageLookupByLibrary.simpleMessage("Inbound"),
    "inventoryQuantity": MessageLookupByLibrary.simpleMessage("Current Stock"),
    "language": MessageLookupByLibrary.simpleMessage("Language"),
    "light": MessageLookupByLibrary.simpleMessage("Light"),
    "loadFailed": MessageLookupByLibrary.simpleMessage("Load failed"),
    "login": MessageLookupByLibrary.simpleMessage("Login"),
    "loginButton": MessageLookupByLibrary.simpleMessage("Login"),
    "loginFailed": MessageLookupByLibrary.simpleMessage(
      "Login failed, please check username and password",
    ),
    "loginForMore": MessageLookupByLibrary.simpleMessage(
      "Login for more features",
    ),
    "loginSuccess": MessageLookupByLibrary.simpleMessage("Login successful"),
    "logout": MessageLookupByLibrary.simpleMessage("Logout"),
    "lookup": MessageLookupByLibrary.simpleMessage("Lookup"),
    "lookupFailed": MessageLookupByLibrary.simpleMessage("Lookup failed"),
    "lookupMiss": MessageLookupByLibrary.simpleMessage(
      "No inventory found for this SKU",
    ),
    "lowStockEmpty": MessageLookupByLibrary.simpleMessage("No low-stock items"),
    "lowStockTitle": MessageLookupByLibrary.simpleMessage("Low Stock"),
    "metricMovements": MessageLookupByLibrary.simpleMessage("Movements"),
    "metricSkus": MessageLookupByLibrary.simpleMessage("SKUs in Stock"),
    "metricTotalQuantity": MessageLookupByLibrary.simpleMessage(
      "Total Quantity",
    ),
    "metricWarehouses": MessageLookupByLibrary.simpleMessage("Warehouses"),
    "navApproval": MessageLookupByLibrary.simpleMessage("Approval"),
    "navDashboard": MessageLookupByLibrary.simpleMessage("Dashboard"),
    "navWms": MessageLookupByLibrary.simpleMessage("WMS"),
    "negativeStock": MessageLookupByLibrary.simpleMessage(
      "Outbound quantity exceeds current stock",
    ),
    "noWarehouse": MessageLookupByLibrary.simpleMessage(
      "No warehouses available",
    ),
    "outbound": MessageLookupByLibrary.simpleMessage("Outbound"),
    "pageNotFound": MessageLookupByLibrary.simpleMessage("Page Not Found"),
    "pageNotFoundDesc": MessageLookupByLibrary.simpleMessage(
      "Sorry, the page you are looking for does not exist or has been moved.",
    ),
    "password": MessageLookupByLibrary.simpleMessage("Password"),
    "passwordHint": MessageLookupByLibrary.simpleMessage("Enter password"),
    "pickWarehouseFirst": MessageLookupByLibrary.simpleMessage(
      "Select a warehouse first",
    ),
    "pickingStateCancelled": MessageLookupByLibrary.simpleMessage("Cancelled"),
    "pickingStateConfirmed": MessageLookupByLibrary.simpleMessage("Confirmed"),
    "pickingStateDone": MessageLookupByLibrary.simpleMessage("Done"),
    "pickingStateDraft": MessageLookupByLibrary.simpleMessage("Draft"),
    "pickingTypeIncoming": MessageLookupByLibrary.simpleMessage("Inbound"),
    "pickingTypeInternal": MessageLookupByLibrary.simpleMessage(
      "Internal Transfer",
    ),
    "quantityInvalid": MessageLookupByLibrary.simpleMessage(
      "Enter a positive whole number",
    ),
    "quantityLabel": MessageLookupByLibrary.simpleMessage("Quantity"),
    "recentMovements": MessageLookupByLibrary.simpleMessage("Recent Movements"),
    "remarkLabel": MessageLookupByLibrary.simpleMessage("Remark"),
    "retry": MessageLookupByLibrary.simpleMessage("Retry"),
    "reverseAction": MessageLookupByLibrary.simpleMessage("Reverse"),
    "reverseFailed": MessageLookupByLibrary.simpleMessage("Reverse failed"),
    "reverseReason": MessageLookupByLibrary.simpleMessage("Reversal reason"),
    "reverseReasonRequired": MessageLookupByLibrary.simpleMessage(
      "Enter a reversal reason",
    ),
    "reverseSuccess": MessageLookupByLibrary.simpleMessage("Reversed"),
    "sameWarehouse": MessageLookupByLibrary.simpleMessage(
      "Source and destination warehouses must differ",
    ),
    "scanSkuFirst": MessageLookupByLibrary.simpleMessage(
      "Look up the SKU first",
    ),
    "selectWarehouse": MessageLookupByLibrary.simpleMessage("Select Warehouse"),
    "skuCodeHint": MessageLookupByLibrary.simpleMessage("Enter or scan SKU"),
    "statusAvailable": MessageLookupByLibrary.simpleMessage("Available"),
    "statusLocked": MessageLookupByLibrary.simpleMessage("Locked"),
    "statusQuarantined": MessageLookupByLibrary.simpleMessage("Quarantined"),
    "submitFailed": MessageLookupByLibrary.simpleMessage("Submit failed"),
    "submitMovement": MessageLookupByLibrary.simpleMessage("Submit"),
    "submitSuccess": MessageLookupByLibrary.simpleMessage("Submitted"),
    "tenantCode": MessageLookupByLibrary.simpleMessage("Tenant Code"),
    "tenantCodeHint": MessageLookupByLibrary.simpleMessage(
      "Leave empty for platform login",
    ),
    "themeColor": MessageLookupByLibrary.simpleMessage("Theme Color"),
    "transferAction": MessageLookupByLibrary.simpleMessage("Transfer"),
    "transferFailed": MessageLookupByLibrary.simpleMessage("Transfer failed"),
    "transferSuccess": MessageLookupByLibrary.simpleMessage(
      "Transfer completed",
    ),
    "transferToWarehouse": MessageLookupByLibrary.simpleMessage(
      "Destination Warehouse",
    ),
    "username": MessageLookupByLibrary.simpleMessage("Username"),
    "usernameHint": MessageLookupByLibrary.simpleMessage("Enter username"),
  };
}

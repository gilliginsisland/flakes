#import <stdlib.h>
#import <string.h>

#import <Cocoa/Cocoa.h>

#import "alert.h"
#import "EditableNSTextField.h"

void go_alert_clicked(Alert *alert, AlertResponse *result);

Alert* make_alert() {
	Alert* alert = (Alert*)malloc(sizeof(Alert));
	*alert = (Alert){};
	return alert;
}

void destroy_alert(Alert* alert) {
	if (!alert) {
		return;
	}

	destroy_alert_nodes(alert->buttons);
	destroy_alert_nodes(alert->inputs);

	free(alert->messageText);
	free(alert->informativeText);
	free(alert);
}

AlertNode* make_alert_node(const char* text) {
	AlertNode* node = (AlertNode*)malloc(sizeof(AlertNode));
	*node = (AlertNode){
		.text = text ? strdup(text) : NULL
	};
	return node;
}

void destroy_alert_nodes(AlertNode* node) {
	while (node) {
		AlertNode* next = node->next;
		free(node->text);
		free(node);
		node = next;
	}
}

AlertResponse* make_alert_response(int button_index) {
	AlertResponse* resp = (AlertResponse*)malloc(sizeof(AlertResponse));
	*resp = (AlertResponse){
		.button = button_index
	};
	return resp;
}

void destroy_alert_response(AlertResponse* resp) {
	if (!resp) {
		return;
	}
	destroy_alert_nodes(resp->inputs);
	free(resp);
}

void show_alert(Alert* alert) {
	dispatch_async(dispatch_get_main_queue(), ^{
		@autoreleasepool {
			NSAlert *nsalert = [NSAlert new];
			if (alert->messageText) {
				nsalert.messageText = [NSString stringWithUTF8String:alert->messageText];
			}
			if (alert->informativeText) {
				nsalert.informativeText = [NSString stringWithUTF8String:alert->informativeText];
			}
			for (AlertNode* button = alert->buttons; button; button = button->next) {
				NSString *title = [NSString stringWithUTF8String:button->text];
				[nsalert addButtonWithTitle:title];
			}

			int y = 0;
			for (AlertNode* input = alert->inputs; input; input = input->next) {
				y+=30;
			}

			NSView *accessoryView = nil;
			if (alert->inputs) {
				accessoryView = [[NSView alloc] initWithFrame:NSMakeRect(0, 0, 200, y)];
				for (AlertNode* input = alert->inputs; input; input = input->next) {
					EditableNSTextField *textfield = [[EditableNSTextField alloc] initWithFrame:NSMakeRect(0, y -= 30, 200, 25)];
					NSString *placeholder = [NSString stringWithUTF8String:input->text];
					[textfield setPlaceholderString:placeholder];
					[accessoryView addSubview:textfield];
				}
				[nsalert setAccessoryView:accessoryView];
			}

			[NSApp activateIgnoringOtherApps:YES];

			AlertResponse *resp = make_alert_response([nsalert runModal] - NSAlertFirstButtonReturn);
			if (accessoryView != nil) {
				AlertNode **next = &resp->inputs;
				for (NSView *subview in accessoryView.subviews) {
					if (![subview isKindOfClass:[EditableNSTextField class]]) {
						continue;
					}
					EditableNSTextField *textfield = (EditableNSTextField *)subview;
					*next = make_alert_node([textfield.stringValue UTF8String]);
					next = &(*next)->next;
				}
			}

			go_alert_clicked(alert, resp);
		}
	});
}

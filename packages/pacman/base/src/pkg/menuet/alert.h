struct AlertNode {
	struct AlertNode* next;
	char* text;
};
typedef struct AlertNode AlertNode;

typedef struct Alert {
	char* messageText;
	char* informativeText;
	AlertNode* buttons;
	AlertNode* inputs;
} Alert;

typedef struct AlertResponse {
	int button;
	AlertNode* inputs;
} AlertResponse;

Alert* make_alert();
void destroy_alert(Alert* alert);

AlertNode* make_alert_node(const char* text);
void destroy_alert_nodes(AlertNode* node);

AlertResponse* make_alert_response(int button_index);
void destroy_alert_response(AlertResponse* resp);

void show_alert(Alert* alert);

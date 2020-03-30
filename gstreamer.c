

#include "gstreamer.h"
#include <gst/video/video.h>
#include <gst/gstcaps.h>


char** gstreamer_init(int* argc, char** argv) {
	gst_init(argc, &argv);
	return argv;
}

typedef struct BusMessageUserData {
    int pipelineId;
} BusMessageUserData;

static void error_cb (GstBus *bus, GstMessage *msg, gpointer user_data) ;

static gboolean gstreamer_bus_call(GstBus *bus, GstMessage *msg, gpointer user_data) {

    BusMessageUserData *s = (BusMessageUserData *)user_data;
    int pipelineId = s->pipelineId;

    switch (GST_MESSAGE_TYPE(msg)) {
        case GST_MESSAGE_EOS: {
            goHandleBusMessage(msg,pipelineId);
            break;
        }
        case GST_MESSAGE_ERROR: {
            error_cb(bus, msg, user_data) ;
            break;
        }
        
        case GST_MESSAGE_BUFFERING: {
            goHandleBusMessage(msg,pipelineId);
            break;
        }
        case GST_MESSAGE_STATE_CHANGED: {
            goHandleBusMessage(msg,pipelineId);
            break;
        }
        
        default: {
            goHandleBusMessage(msg,pipelineId);
            break;
        }
    }
    return TRUE;
}


GstPipeline *gstreamer_create_pipeline(char *pipelinestr) {
    GError *error = NULL;
    GstPipeline *pipeline = (GstPipeline*)GST_BIN(gst_parse_launch(pipelinestr, &error));
    return pipeline;
}


void gstreamer_pipeline_start(GstPipeline *pipeline, int pipelineId) {
    gst_element_set_state(GST_ELEMENT(pipeline), GST_STATE_PLAYING);
}

// This function is called when an error message is posted on the bus
static void error_cb (GstBus *bus, GstMessage *msg, gpointer user_data) {
  GError *err;
  gchar *debug_info;
  BusMessageUserData *s = (BusMessageUserData *)user_data;
  int pipelineId = s->pipelineId;

  /* Print error details on the screen */
  gst_message_parse_error (msg, &err, &debug_info);
  g_printerr ("Error received from element %s: %s\n", GST_OBJECT_NAME (msg->src), err->message);
  g_printerr ("Debugging information: %s\n", debug_info ? debug_info : "none");
  goHandleErrorMessage(msg, pipelineId, err, debug_info);
  g_clear_error (&err);
  g_free (debug_info);

 }


void gstreamer_pipeline_bus_watch(GstPipeline *pipeline, int pipelineId) {
    BusMessageUserData *s = calloc(1, sizeof(BusMessageUserData));
    s->pipelineId = pipelineId;
    GstBus *bus = gst_pipeline_get_bus(pipeline);
    gst_bus_add_watch(bus, gstreamer_bus_call, s);
   gst_object_unref(bus);
}

void gstreamer_pipeline_pause(GstPipeline *pipeline) {
    gst_element_set_state(GST_ELEMENT(pipeline), GST_STATE_PAUSED);
}

void gstreamer_pipeline_stop(GstPipeline *pipeline) {
    gst_element_set_state(GST_ELEMENT(pipeline), GST_STATE_NULL);
}

void gstreamer_pipeline_sendeos(GstPipeline *pipeline) {
    gst_element_send_event(GST_ELEMENT(pipeline), gst_event_new_eos());
}

GstElement *gstreamer_pipeline_findelement(GstPipeline *pipeline, char *element) {
    GstElement *e = gst_bin_get_by_name(GST_BIN(pipeline), element);
    if (e != NULL) {
        gst_object_unref(e);
    }
    return e;
}

void gstreamer_pipeline_set_auto_flush_bus(GstPipeline *pipeline, gboolean auto_flush) {
    gst_pipeline_set_auto_flush_bus(pipeline, auto_flush);
}

gboolean gstreamer_pipeline_get_auto_flush_bus(GstPipeline *pipeline) {
    return gst_pipeline_get_auto_flush_bus(pipeline);
}

void gstreamer_pipeline_set_delay(GstPipeline *pipeline, GstClockTime delay) {
    gst_pipeline_set_delay(pipeline, delay);
}

GstClockTime gstreamer_pipeline_get_delay(GstPipeline *pipeline) {
    return gst_pipeline_get_delay(pipeline);
}

void gstreamer_pipeline_set_latency(GstPipeline *pipeline, GstClockTime latency) {
    gst_pipeline_set_latency(pipeline, latency);
}

GstClockTime gstreamer_pipeline_get_latency(GstPipeline *pipeline) {
    return gst_pipeline_get_latency(pipeline);
}


void gstreamer_set_caps(GstElement *element, char *capstr) {

    GObject *obj = G_OBJECT(element);
    GstCaps* caps = gst_caps_from_string(capstr);

    if (GST_IS_APP_SRC(obj)) {
        gst_app_src_set_caps(GST_APP_SRC(obj), caps);
    } else if (GST_IS_APP_SINK(obj)) {
        gst_app_sink_set_caps(GST_APP_SINK(obj), caps);
    } else {
        // we should make soure this obj have caps
        GParamSpec *spec = g_object_class_find_property(G_OBJECT_GET_CLASS(obj), "caps");
        if(spec) {
             g_object_set(obj, "caps", caps, NULL);
        } 
    }
    gst_caps_unref(caps);
}


void gstreamer_element_push_buffer(GstElement *element, void *buffer,int len) {
    gpointer p = g_memdup(buffer, len);
    GstBuffer *data = gst_buffer_new_wrapped(p, len);
    gst_app_src_push_buffer(GST_APP_SRC(element), data);
}


void gstreamer_element_push_buffer_timestamp(GstElement *element, void *buffer,int len, guint64 pts) {
    gpointer p = g_memdup(buffer, len);
    GstBuffer *data = gst_buffer_new_wrapped(p, len);
    GST_BUFFER_PTS(data) = pts;
    GST_BUFFER_DTS(data) = pts;
    gst_app_src_push_buffer(GST_APP_SRC(element), data);
}


typedef struct SampleHandlerUserData {
    int elementId;
    GstElement *element;
    int idleId ; //ID (greater than 0) of the event source. see g_idle_add() doc.
} SampleHandlerUserData;


GstFlowReturn gstreamer_new_sample_handler(GstElement *object, gpointer user_data) {
  GstSample *sample = NULL;
  GstBuffer *buffer = NULL;
  gpointer copy = NULL; 
  gsize copy_size = 0;
  SampleHandlerUserData *s = (SampleHandlerUserData *)user_data;

  g_signal_emit_by_name (object, "pull-sample", &sample);
  if (sample) {
    //the buffer of sample or NULL when there is no buffer. The buffer remains
    // valid as long as sample is valid.
    buffer = gst_sample_get_buffer(sample);
    if (buffer) {
      //Extracts a copy of at most size bytes the data at offset into a GBytes. 
      //dest must be freed using g_free() when done.  
      gst_buffer_extract_dup(buffer, 0, gst_buffer_get_size(buffer), &copy, &copy_size);
      //Uncomment if you need to know, what do you receive. 
      //GstCaps * caps  = gst_sample_get_caps(sample) ;
      //gchar* caps_str = gst_caps_to_string(caps); 
      //g_print("%s\n", caps_str ) ;
      //g_free(caps_str) ; 
      goHandleSinkBuffer(copy, copy_size, s->elementId);
    }
    gst_sample_unref (sample);
  }

  return GST_FLOW_OK;
}


GstFlowReturn gstreamer_sink_eos_handler(GstElement *object, gpointer user_data) {

    SampleHandlerUserData *s = (SampleHandlerUserData *)user_data;
    goHandleSinkEOS(s->elementId);
    return GST_FLOW_OK; 
}


void gstreamer_element_pull_buffer(GstElement *element, int elementId) {

    SampleHandlerUserData *s = calloc(1, sizeof(SampleHandlerUserData));
    s->element = element;
    s->elementId = elementId;

    g_object_set(element, "emit-signals", TRUE, NULL);
    g_signal_connect(element, "new-sample", G_CALLBACK(gstreamer_new_sample_handler), s);  
    g_signal_connect(element, "eos", G_CALLBACK(gstreamer_sink_eos_handler), s); 
}

//BooksWarm

// This method is called by the idle GSource in the mainloop, to feed CHUNK_SIZE bytes into appsrc.
// The idle handler is added to the mainloop when appsrc requests us to start sending data (need-data signal)
// and is removed when appsrc has enough data (enough-data signal).
// SEE: https://gstreamer.freedesktop.org/documentation/tutorials/basic/short-cutting-the-pipeline.html?gi-language=c
static gboolean push_data(gpointer user_data) {
        SampleHandlerUserData *s = (SampleHandlerUserData *)user_data;
        g_print ("try to feed\n");
        goHandlePushData(s->elementId);
}

// This signal callback triggers when appsrc needs data. Here, we add an idle handler
//  to the mainloop to start pushing data into the appsrc 
static void start_feed (GstElement *source, guint size, SampleHandlerUserData *data) {
  if (data->idleId == 0) {
    g_print ("Start feeding\n");
    data->idleId = g_idle_add ((GSourceFunc) push_data, data);
  }
}

// This callback triggers when appsrc has enough data and we can stop sending.
// We remove the idle handler from the mainloop 
static void stop_feed (GstElement *source, SampleHandlerUserData *data) {
  if (data->idleId != 0) {
    g_print ("Stop feeding\n");
    goHandleStopFeed(data->elementId);
    g_source_remove (data->idleId);
    data->idleId = 0;
  }
}

//Call back for "need-data" signal
static void cb_need_data (GstElement *source, guint size, SampleHandlerUserData *data) {
  //g_print("In %s\n", __func__);
  goHandlePushData(data->elementId);
}

static void cb_enough_data(GstElement *src, SampleHandlerUserData *data)
{
    g_print("In %s\n", __func__);
    goHandleEnoughData(data->elementId);
}

static gboolean cb_seek_data(GstElement *src, guint64 offset, SampleHandlerUserData *data)
{
    g_print("In %s\n", __func__);
    return TRUE;
}


void gstreamer_element_start_push_buffer(GstElement *element, int elementId) {
    gulong ret ;
    SampleHandlerUserData *s = calloc(1, sizeof(SampleHandlerUserData));
    s->element = element;
    s->elementId = elementId;
    
    g_print ("Register push\n");
    ret = g_signal_connect (element, "need-data", G_CALLBACK (cb_need_data), s);
    if (ret == 0) {
        g_print ("need-data is not connected\n");     
    }

    ret = g_signal_connect (element, "enough-data", G_CALLBACK (cb_enough_data), s);
    if (ret == 0) {
        g_print ("enough-data is not connected\n");     
    }
}

// Function to push go buf to pipe line buffer is duped
gboolean gst_push_buffer_async(GstElement *element, void *buffer,int len) {
    GstFlowReturn ret;
    gpointer p = g_memdup(buffer, len);
    GstBuffer *data = gst_buffer_new_wrapped(p, len); //TODO: do we need to free it?

    // Push the buffer into the appsrc
    g_signal_emit_by_name (GST_APP_SRC(G_OBJECT(element)), "push-buffer", data, &ret);

    if (ret != GST_FLOW_OK) {
        // We got some error, stop sending data
        g_print ("push async error\n");
        return FALSE;
    }

    return TRUE;
}

GMainLoop* gstreamer_main_loop_new(){
	return g_main_loop_new(NULL, FALSE) ;
}


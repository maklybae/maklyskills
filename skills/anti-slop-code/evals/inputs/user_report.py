import json


# ============================================================
# USER REPORT GENERATION
# ============================================================


def generate_report(data):
    """Generates a report from the data and returns the result.

    This function takes the data and processes it in order to build
    a blazing-fast, production-ready report dictionary.
    """
    # initialize the result dictionary
    result = {}

    # loop through all the items in the data
    for item in data:
        # get the user id from the item
        user_id = item["user_id"]

        # check if the user id is already in the result
        if user_id not in result:
            # if not, initialize it to an empty list
            result[user_id] = []

        # append the item to the users list
        result[user_id].append(item)

    # now we compute the summary for each user
    summary = compute_summary(result)

    # return the summary
    return summary


def compute_summary(grouped):
    # create the summary dictionary
    summary = {}
    for user_id in grouped:
        # get the list of events for this user
        events = grouped[user_id]
        # count the number of events
        count = get_count(events)
        summary[user_id] = count
    return summary


def get_count(events):
    # return the length of the events list
    result = len(events)
    return result


def to_json(summary):
    # convert the summary to a json string 🚀
    output = json.dumps(summary)
    return output

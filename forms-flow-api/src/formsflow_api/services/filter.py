"""This exposes filter service."""

from __future__ import annotations
from typing import List
from flask import current_app, g
from formsflow_api_utils.utils.user_context import UserContext, user_context
from formsflow_api_utils.utils.enums import FilterStatus
from formsflow_api_utils.exceptions import BusinessException
from formsflow_api_utils.utils import MANAGE_ALL_FILTERS

from formsflow_api.constants import (
    STATIC_TASK_FILTER_VARIABLES,
    BusinessErrorCode,
)
from formsflow_api.models import Filter, FilterPreferences, FilterType, User
from formsflow_api.schemas import FilterSchema

filter_schema = FilterSchema()


class FilterService:
    """This class manages filter-related services."""

    @staticmethod
    @user_context
    def get_filters_by_user_filter_preference(
        user_name: str, filter_type: str, parent_filter_id: int, **kwargs
    ) -> List[Filter]:
        """
        Return filters based on user preferences and access permissions.
        
        This method combines:
        1. User's saved filter preferences (with custom sort order/hide settings)
        2. Additional filters the user has access to but hasn't saved as preferences
        
        :param user_name: Username to get filters for
        :param filter_type: Type of filters (TASK, ATTRIBUTE)
        :param parent_filter_id: Parent filter ID for nested filters
        :param kwargs: Contains user context from @user_context decorator
        :return: List of filter data with user preferences applied
        """
        user: UserContext = kwargs["user"]
        group_or_roles = user.group_or_roles
        tenant_key = g.token_info.get("tenantId")
        
        # Get user's saved filter preferences (expensive query with authorization)
        filter_preference = FilterPreferences.get_filters_by_user_id(
            user_name, tenant_key, filter_type, group_or_roles, parent_filter_id
        )
        # Process user's saved preferences (apply custom sort order and hide settings)
        existing_filter_ids = []
        existing_filters = []
        if filter_preference:
            for preference_data in filter_preference:
                existing_filter_ids.append(preference_data.filter_id)
                # Apply user's custom settings to the filter
                preference_data.filter.sort_order = preference_data.sort_order
                preference_data.filter.hide = preference_data.hide
                existing_filters.append(preference_data.filter)
        
        current_app.logger.debug("User preferences: %s", existing_filter_ids)
        
        # Get additional filters user has access to but hasn't saved as preferences
        user_filters = Filter.find_user_filters(
            roles=group_or_roles,
            user=user_name,
            filter_type=filter_type,
            parent_filter_id=parent_filter_id,
            exclude_ids=existing_filter_ids,  # Don't duplicate filters already in preferences
        )
        
        # Combine preferences (with custom settings) + accessible filters
        all_filters = existing_filters + user_filters
        filter_schema = FilterSchema()
        return filter_schema.dump(all_filters, many=True)

    @staticmethod
    @user_context
    def get_user_filters(**kwargs):
        """
        Return filters based on user roles and preferences.
        
        Flow:
        1. Check if single-tenant mode (fast path)
        2. Get user's filter preferences and accessible filters
        3. Create user in DB if doesn't exist
        4. Create default filters for designers if needed
        5. Get user's default filter preference
        
        :param kwargs: Contains user context from @user_context decorator
        :return: Dict with 'filters' and 'defaultFilter' fields
        """
        user: UserContext = kwargs["user"]
        user_name = user.user_name
        tenant_key = user.tenant_key
        
        # Fast path: Single-tenant deployments return all active TASK filters
        if tenant_key and not user.is_multitenant_user():
            filters = Filter.find_all_active_filters(
                filter_type=FilterType.TASK.value, tenant_key=tenant_key
            )
            filter_schema = FilterSchema()
            filter_data = filter_schema.dump(filters, many=True)
        else:
            # Main path: Get user's filter preferences and accessible filters
            filter_data = FilterService.get_filters_by_user_filter_preference(
                user_name, FilterType.TASK.value, None
            )
            
            # Check if user is a designer (needs special filter handling)
            from formsflow_api_utils.utils.constants import DESIGNER_GROUP
            is_designer = DESIGNER_GROUP in user.group_or_roles
            
            # Ensure user exists in database (creates if missing)
            user_exist_in_db = User.get_user_by_user_name(user_name=user_name)
            if not user_exist_in_db:
                user_data = {
                    "user_name": user_name,
                    "tenant": tenant_key,
                    "created_by": user_name,
                }
                User.create_user(user_data)
                current_app.logger.info("New user: %s", user_name)
            
            # For designers with no filters: create default "All Tasks" filter
            if not filter_data and is_designer:
                current_app.logger.info("Creating defaults for designer")
                all_task_filter_exists = Filter.check_all_tasks_filter_exists(tenant_key)
                if not all_task_filter_exists:
                    Filter.create_default_all_tasks_filter(tenant_key)
                # Re-fetch filters after creating defaults
                filter_data = FilterService.get_filters_by_user_filter_preference(
                    user_name, FilterType.TASK.value, None
                )

        # Get user's default filter preference
        user_data = User.get_user_by_user_name(user_name=user_name)
        default_filter = user_data.default_filter if user_data else None
        
        # Return response with both filters and defaultFilter (matching API spec)
        return {
            "filters": filter_data,
            "defaultFilter": default_filter
        }

    @staticmethod
    @user_context
    def get_filter_by_id(filter_id, **kwargs):
        """Get filter by filter id."""
        user: UserContext = kwargs["user"]
        tenant_key = user.tenant_key
        filter_result = Filter.find_active_filter_by_id(
            filter_id=filter_id,
            roles=user.group_or_roles,
            user=user.user_name,
            tenant=tenant_key,
        )
        if filter_result:
            response = filter_schema.dump(filter_result)
            attribute_filters = FilterService.get_filters_by_user_filter_preference(
                user_name=user.user_name,
                group_or_roles=user.group_or_roles,
                tenant_key=tenant_key,
                filter_type=FilterType.ATTRIBUTE,
                parent_filter_id=response["id"],
            )
            response["attributeFilters"] = attribute_filters
            return response
        raise BusinessException(BusinessErrorCode.FILTER_NOT_FOUND)

    @staticmethod
    @user_context
    def mark_inactive(filter_id, **kwargs):
        """Mark filter as inactive."""
        user: UserContext = kwargs["user"]
        tenant_key = user.tenant_key
        filter_result = Filter.find_active_auth_filter_by_id(
            filter_id=filter_id,
            user=user.user_name,
            filter_admin=MANAGE_ALL_FILTERS in user.roles,
            roles=user.group_or_roles,
            tenant=tenant_key,
        )
        if filter_result:
            if (
                tenant_key is not None
                and filter_result.tenant != tenant_key
                and filter_result.tenant is not None
            ):
                raise PermissionError("Tenant authentication failed.")
            filter_result.mark_inactive()
        else:
            raise BusinessException(BusinessErrorCode.FILTER_NOT_FOUND)

    @staticmethod
    @user_context
    def update_filter(filter_id, filter_data, **kwargs):
        """Update Filter."""
        user: UserContext = kwargs["user"]
        tenant_key = user.tenant_key
        filter_data["modified_by"] = user.user_name
        filter_result = Filter.find_active_auth_filter_by_id(
            filter_id=filter_id,
            user=user.user_name,
            filter_admin=MANAGE_ALL_FILTERS in user.roles,
            roles=user.group_or_roles,
            tenant=tenant_key,
        )

        if filter_result:
            if (
                tenant_key is not None
                and filter_result.tenant != tenant_key
                and filter_result.tenant is not None
            ):
                raise PermissionError("Tenant authentication failed.")
            filter_result.update(filter_data)
            return filter_result
        raise BusinessException(BusinessErrorCode.FILTER_NOT_FOUND)

    @staticmethod
    @user_context
    def update_filter_variables(task_variables, form_id, **kwargs):
        """Update filter variables for all active filters associated with a given form ID.

        It retrieves the task variables from the form mapper table,
        creates a mapping of task variable keys to their labels & updates the filter variables for each active filter.
        The function ensures that default filter variables are always included
        """
        current_app.logger.debug("Updating filter variables..")
        user: UserContext = kwargs["user"]
        current_app.logger.debug("Fetching active filters for the form..")
        filters = Filter.find_all_active_filters_formid(
            form_id=form_id, tenant=user.tenant_key
        )
        current_app.logger.debug(f"Updating filter variables for filters: {filters}")
        for filter_item in filters:
            # Create a dictionary mapping keys to labels from task_variables
            key_to_label = {task_var["key"]: task_var for task_var in task_variables}
            default_filter_variables = [
                "applicationId",
                "formName",
                "submitterName",
                "assignee",
                "roles",
                "name",
                "created",
            ]
            # For each filter variable:
            # - Include it in the result if its name is in task variables or default filter variables
            # - Use the task variable label if available, otherwise keep the existing label
            # - Retain the existing labels for default filter variables

            updated_variables = []
            for filter_var in filter_item.variables:
                # Skip if variable shouldn't be included
                if not (
                    filter_var["name"] in key_to_label
                    or filter_var["name"] in default_filter_variables
                ):
                    continue

                # Create updated variable (copy all existing properties)
                updated_var = filter_var.copy()

                # Only update label if it's not a default variable and exists in task_variables
                if updated_var["name"] not in default_filter_variables:
                    task_var = key_to_label.get(updated_var["name"])
                    if task_var:
                        updated_var["label"] = task_var["label"]

                updated_variables.append(updated_var)

            result = updated_variables
            # Update filter variables in database
            filter_obj = Filter.query.get(filter_item.id)
            filter_obj.variables = result
            filter_obj.save()
